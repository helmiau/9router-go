package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"9router/proxy/internal/db"
	"9router/proxy/internal/handlerutil"
)

// GitHub import for the jailbreak preset library. An operator points the
// dashboard at a public repository of prompts — one file per model, the shape
// collections such as togg53192-cmd/jailbreaks use — and every file becomes a
// preset whose keywords are derived from its name, so it covers that model
// across providers and combo legs without arming each id by hand.
//
// The target is operator input, so only GitHub hosts are reachable: a fetcher
// that followed it anywhere would turn the dashboard into a proxy for the
// operator's own network. Redirects are re-checked against the same allowlist,
// every response is size-bounded, and the file count and total bytes are
// capped before anything is downloaded.

// Bounds for one import. A collection is a few dozen files of a few tens of
// kilobytes; the caps keep a mistyped repository from becoming a long download
// on a one-vCPU box.
const (
	maxJailbreakImportFiles      = 200
	maxJailbreakImportFileBytes  = 512 * 1024
	maxJailbreakImportTotalBytes = 8 * 1024 * 1024
	jailbreakImportParallel      = 4
	jailbreakImportTimeout       = 60 * time.Second
)

// maxJailbreakImportKeywords mirrors the storage bound in
// db.NormalizeJailbreakKeywords, which rejects a preset carrying more. The
// derivation caps itself so an import never produces a preset the store
// refuses.
const maxJailbreakImportKeywords = 20

// errJailbreakBadTarget marks a target the operator typed wrong. The handler
// answers 400 for it and 502 for anything the network did.
var errJailbreakBadTarget = errors.New("invalid github target")

var (
	githubNameRE = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	// A ref is one or more slash-separated names: no leading, trailing or
	// empty segment, so "main", "feature/x" and a sha are fine while
	// "/etc/passwd" and "a//b" are not.
	githubRefRE = regexp.MustCompile(`^[A-Za-z0-9._-]+(/[A-Za-z0-9._-]+)*$`)
)

type jailbreakImporter struct {
	apiBase string
	rawBase string
	client  *http.Client
}

var defaultJailbreakImporter = newJailbreakImporter()

// newJailbreakImporter builds the production importer. One client serves the
// whole import, with a redirect policy that keeps every hop on GitHub.
func newJailbreakImporter() *jailbreakImporter {
	return &jailbreakImporter{
		apiBase: "https://api.github.com",
		rawBase: "https://raw.githubusercontent.com",
		client: &http.Client{
			Timeout: jailbreakImportTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}
				if !isGithubHost(req.URL.Hostname()) {
					return fmt.Errorf("redirect to non-GitHub host %q refused", req.URL.Hostname())
				}
				return nil
			},
		},
	}
}

func isGithubHost(host string) bool {
	switch strings.ToLower(host) {
	case "github.com", "www.github.com", "raw.githubusercontent.com", "api.github.com":
		return true
	}
	return false
}

// githubJailbreakSource is a parsed import target. An empty Path means the
// whole repository; a non-empty one means a single file.
type githubJailbreakSource struct {
	Owner string
	Repo  string
	Ref   string
	Path  string
}

// githubFile is one prompt file to fetch.
type githubFile struct {
	Path string
	Size int
}

// githubTreeEntry is one blob in a repository tree.
type githubTreeEntry struct {
	Path string
	Size int
}

// jailbreakImportEntry is one preset the import wrote.
type jailbreakImportEntry struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Keywords []string `json:"keywords"`
}

// jailbreakImportRefusal is one file the import skipped, with the reason, so a
// partial import never reads as a complete one.
type jailbreakImportRefusal struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type jailbreakImportResult struct {
	Source   string                   `json:"source"`
	Imported []jailbreakImportEntry   `json:"imported"`
	Refused  []jailbreakImportRefusal `json:"refused"`
}

// parseGithubJailbreakTarget accepts owner/repo, owner/repo@ref, and the
// github.com, raw.githubusercontent.com and api.github.com URL forms.
func parseGithubJailbreakTarget(target string) (githubJailbreakSource, error) {
	t := strings.TrimSpace(target)
	if t == "" {
		return githubJailbreakSource{}, fmt.Errorf("%w: empty target", errJailbreakBadTarget)
	}
	if !strings.Contains(t, "://") {
		return parseGithubShorthand(t)
	}
	u, err := url.Parse(t)
	if err != nil {
		return githubJailbreakSource{}, fmt.Errorf("%w: %v", errJailbreakBadTarget, err)
	}
	switch strings.ToLower(u.Hostname()) {
	case "github.com", "www.github.com":
		return parseGithubWebPath(u.Path)
	case "raw.githubusercontent.com":
		return parseGithubRawPath(u.Path)
	case "api.github.com":
		return parseGithubAPIPath(u.Path)
	default:
		return githubJailbreakSource{}, fmt.Errorf("%w: host %q is not GitHub", errJailbreakBadTarget, u.Hostname())
	}
}

// parseGithubShorthand reads "owner/repo" and "owner/repo@ref". A path needs a
// URL: three bare segments would be ambiguous with a branch name.
func parseGithubShorthand(t string) (githubJailbreakSource, error) {
	ref := ""
	if at := strings.Index(t, "@"); at >= 0 {
		ref = t[at+1:]
		t = t[:at]
	}
	parts := strings.Split(t, "/")
	if len(parts) != 2 {
		return githubJailbreakSource{}, fmt.Errorf("%w: want owner/repo, got %q", errJailbreakBadTarget, t)
	}
	src := githubJailbreakSource{Owner: parts[0], Repo: parts[1], Ref: ref}
	if err := src.validate(); err != nil {
		return githubJailbreakSource{}, err
	}
	return src, nil
}

// parseGithubWebPath reads /owner/repo, /owner/repo/tree/ref/path and
// /owner/repo/blob/ref/path.
func parseGithubWebPath(p string) (githubJailbreakSource, error) {
	segs := splitGithubPath(p)
	if len(segs) < 2 {
		return githubJailbreakSource{}, fmt.Errorf("%w: want github.com/owner/repo", errJailbreakBadTarget)
	}
	src := githubJailbreakSource{Owner: segs[0], Repo: segs[1]}
	switch {
	case len(segs) == 2:
	case len(segs) > 3 && (segs[2] == "tree" || segs[2] == "blob"):
		src.Ref = segs[3]
		src.Path = strings.Join(segs[4:], "/")
	default:
		return githubJailbreakSource{}, fmt.Errorf("%w: unsupported path %q", errJailbreakBadTarget, p)
	}
	if err := src.validate(); err != nil {
		return githubJailbreakSource{}, err
	}
	return src, nil
}

// parseGithubRawPath reads /owner/repo/ref/path, which names one file.
func parseGithubRawPath(p string) (githubJailbreakSource, error) {
	segs := splitGithubPath(p)
	if len(segs) < 4 {
		return githubJailbreakSource{}, fmt.Errorf("%w: want raw.githubusercontent.com/owner/repo/ref/path", errJailbreakBadTarget)
	}
	src := githubJailbreakSource{Owner: segs[0], Repo: segs[1], Ref: segs[2], Path: strings.Join(segs[3:], "/")}
	if err := src.validate(); err != nil {
		return githubJailbreakSource{}, err
	}
	return src, nil
}

// parseGithubAPIPath reads /repos/owner/repo and /repos/owner/repo/git/trees/ref.
func parseGithubAPIPath(p string) (githubJailbreakSource, error) {
	segs := splitGithubPath(p)
	if len(segs) > 0 && segs[0] == "repos" {
		segs = segs[1:]
	}
	if len(segs) < 2 {
		return githubJailbreakSource{}, fmt.Errorf("%w: want api.github.com/repos/owner/repo", errJailbreakBadTarget)
	}
	src := githubJailbreakSource{Owner: segs[0], Repo: segs[1]}
	if len(segs) > 3 && segs[2] == "git" && segs[3] == "trees" && len(segs) > 4 {
		src.Ref = segs[4]
	}
	if err := src.validate(); err != nil {
		return githubJailbreakSource{}, err
	}
	return src, nil
}

func splitGithubPath(p string) []string {
	out := make([]string, 0, 6)
	for _, seg := range strings.Split(p, "/") {
		if seg != "" {
			out = append(out, seg)
		}
	}
	return out
}

// validate keeps the parsed pieces inside the GitHub name grammar and rejects
// anything that could climb out of the repository once joined onto a URL.
func (s githubJailbreakSource) validate() error {
	if !githubNameRE.MatchString(s.Owner) || !githubNameRE.MatchString(s.Repo) {
		return fmt.Errorf("%w: bad owner/repo %q/%q", errJailbreakBadTarget, s.Owner, s.Repo)
	}
	if s.Ref != "" {
		if !githubRefRE.MatchString(s.Ref) || hasGithubDotSegment(s.Ref) {
			return fmt.Errorf("%w: bad ref %q", errJailbreakBadTarget, s.Ref)
		}
	}
	if s.Path != "" {
		if strings.HasPrefix(s.Path, "/") {
			return fmt.Errorf("%w: path %q must be relative", errJailbreakBadTarget, s.Path)
		}
		if hasGithubDotSegment(s.Path) {
			return fmt.Errorf("%w: path %q has a . or .. segment", errJailbreakBadTarget, s.Path)
		}
	}
	return nil
}

func hasGithubDotSegment(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		if seg == "." || seg == ".." {
			return true
		}
	}
	return false
}

// importPresets parses an operator target and imports it.
func (imp *jailbreakImporter) importPresets(ctx context.Context, repo *db.Repo, target string) (*jailbreakImportResult, error) {
	src, err := parseGithubJailbreakTarget(target)
	if err != nil {
		return nil, err
	}
	return imp.importSource(ctx, repo, src)
}

// importSource downloads one source and writes its prompt files as presets.
// Every file is attempted on its own: a single refusal never aborts the rest,
// and the result names what was skipped and why.
func (imp *jailbreakImporter) importSource(ctx context.Context, repo *db.Repo, src githubJailbreakSource) (*jailbreakImportResult, error) {
	files, skipped, ref, err := imp.collectFiles(ctx, src)
	if err != nil {
		return nil, err
	}
	claimed := jailbreakClaimedStems(files)
	res := &jailbreakImportResult{
		Source:   fmt.Sprintf("%s/%s@%s", src.Owner, src.Repo, ref),
		Imported: make([]jailbreakImportEntry, 0, len(files)),
		Refused:  skipped,
	}
	if res.Refused == nil {
		res.Refused = make([]jailbreakImportRefusal, 0)
	}

	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		sem = make(chan struct{}, jailbreakImportParallel)
	)
	for _, f := range files {
		wg.Add(1)
		go func(f githubFile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			entry, refusal := imp.importOne(ctx, repo, src, ref, f, claimed)
			mu.Lock()
			defer mu.Unlock()
			if refusal != nil {
				res.Refused = append(res.Refused, *refusal)
				return
			}
			res.Imported = append(res.Imported, *entry)
		}(f)
	}
	wg.Wait()

	sort.Slice(res.Imported, func(i, j int) bool { return res.Imported[i].Path < res.Imported[j].Path })
	sort.Slice(res.Refused, func(i, j int) bool { return res.Refused[i].Path < res.Refused[j].Path })
	return res, nil
}

// collectFiles lists the repository, or returns the single named file, and
// reports the candidates it dropped for size or count before any download.
func (imp *jailbreakImporter) collectFiles(ctx context.Context, src githubJailbreakSource) ([]githubFile, []jailbreakImportRefusal, string, error) {
	ref := src.Ref
	if ref == "" {
		def, err := imp.defaultBranch(ctx, src)
		if err != nil {
			return nil, nil, "", err
		}
		ref = def
	}
	if src.Path != "" {
		return []githubFile{{Path: src.Path}}, nil, ref, nil
	}

	entries, truncated, err := imp.tree(ctx, src, ref)
	if err != nil {
		return nil, nil, "", err
	}
	if truncated {
		return nil, nil, "", fmt.Errorf("repository %s/%s tree is too large to list", src.Owner, src.Repo)
	}

	files := make([]githubFile, 0, len(entries))
	skipped := make([]jailbreakImportRefusal, 0)
	total := 0
	for _, e := range entries {
		if !jailbreakImportCandidate(e.Path) {
			continue
		}
		switch {
		case e.Size > maxJailbreakImportFileBytes:
			skipped = append(skipped, jailbreakImportRefusal{Path: e.Path, Reason: fmt.Sprintf("file is %d bytes (max %d)", e.Size, maxJailbreakImportFileBytes)})
		case len(files) >= maxJailbreakImportFiles:
			skipped = append(skipped, jailbreakImportRefusal{Path: e.Path, Reason: fmt.Sprintf("import is capped at %d files", maxJailbreakImportFiles)})
		case total+e.Size > maxJailbreakImportTotalBytes:
			skipped = append(skipped, jailbreakImportRefusal{Path: e.Path, Reason: fmt.Sprintf("import is capped at %d bytes total", maxJailbreakImportTotalBytes)})
		default:
			total += e.Size
			files = append(files, githubFile{Path: e.Path, Size: e.Size})
		}
	}
	if len(files) == 0 {
		return nil, nil, "", fmt.Errorf("no .md or .txt prompt files found in %s/%s", src.Owner, src.Repo)
	}
	return files, skipped, ref, nil
}

// jailbreakImportCandidate keeps prompt files and drops repository furniture:
// dot-directories such as .github, and the README that describes a collection
// rather than being one of its prompts.
func jailbreakImportCandidate(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		if strings.HasPrefix(seg, ".") {
			return false
		}
	}
	base := strings.ToLower(path.Base(p))
	if base == "readme.md" || base == "readme.markdown" || base == "readme.txt" {
		return false
	}
	return strings.HasSuffix(base, ".md") || strings.HasSuffix(base, ".txt")
}

// importOne downloads one file and writes it as a preset.
func (imp *jailbreakImporter) importOne(ctx context.Context, repo *db.Repo, src githubJailbreakSource, ref string, f githubFile, claimed map[string]bool) (*jailbreakImportEntry, *jailbreakImportRefusal) {
	text, err := imp.fetchText(ctx, imp.rawURL(src, ref, f.Path), maxJailbreakImportFileBytes)
	if err != nil {
		return nil, &jailbreakImportRefusal{Path: f.Path, Reason: err.Error()}
	}
	if strings.TrimSpace(text) == "" {
		return nil, &jailbreakImportRefusal{Path: f.Path, Reason: "file is empty"}
	}
	stem := jailbreakFileStem(f.Path)
	doc := &db.JailbreakDoc{
		ID:       slugify(stem),
		Name:     stem,
		Prompt:   text,
		Source:   fmt.Sprintf("%s/%s@%s/%s", src.Owner, src.Repo, ref, f.Path),
		Keywords: jailbreakImportKeywords(stem, claimed),
	}
	if err := repo.SaveJailbreakPreset(doc); err != nil {
		return nil, &jailbreakImportRefusal{Path: f.Path, Reason: err.Error()}
	}
	keywords := make([]string, 0, len(doc.Keywords))
	for _, kw := range doc.Keywords {
		keywords = append(keywords, kw.Text)
	}
	return &jailbreakImportEntry{ID: doc.ID, Name: doc.Name, Path: f.Path, Keywords: keywords}, nil
}

// jailbreakClaimedStems is the set of full normalized file names in the
// import, used to stop a shorter rule from stealing another file's model.
func jailbreakClaimedStems(files []githubFile) map[string]bool {
	claimed := make(map[string]bool, len(files))
	for _, f := range files {
		if stem := normalizeJailbreakStem(jailbreakFileStem(f.Path)); stem != "" {
			claimed[stem] = true
		}
	}
	return claimed
}

func jailbreakFileStem(p string) string {
	base := path.Base(p)
	return strings.TrimSuffix(base, path.Ext(base))
}

// jailbreakImportKeywords derives match rules from a file name. "deepseek v4"
// yields the chain deepseek, deepseek-v4: resolution prefers the longest
// matching rule, so the file named after a model beats a sibling that only
// shares the family token, while providerx/deepseek-v4 resolves exactly as
// deepseek-v4 does.
func jailbreakImportKeywords(stem string, claimed map[string]bool) []db.JailbreakKeyword {
	norm := normalizeJailbreakStem(stem)
	if norm == "" {
		return nil
	}
	segs := strings.Split(norm, "-")
	out := make([]db.JailbreakKeyword, 0, len(segs))
	for i := range segs {
		kw := strings.Join(segs[:i+1], "-")
		if len(kw) < 2 {
			continue
		}
		if kw != norm && claimed[kw] {
			continue
		}
		out = append(out, db.JailbreakKeyword{Text: kw})
	}
	if len(out) > maxJailbreakImportKeywords {
		out = out[len(out)-maxJailbreakImportKeywords:]
	}
	return out
}

// normalizeJailbreakStem lowercases a file name and turns separators into the
// hyphens model ids use, keeping dots: "Muse spark 1.3" becomes
// muse-spark-1.3, which is the id a client sends.
func normalizeJailbreakStem(stem string) string {
	var b strings.Builder
	pendingSep := false
	for _, r := range strings.ToLower(strings.TrimSpace(stem)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.':
			if pendingSep && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingSep = false
			b.WriteRune(r)
		default:
			pendingSep = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func (imp *jailbreakImporter) rawURL(src githubJailbreakSource, ref, p string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", imp.rawBase,
		url.PathEscape(src.Owner), url.PathEscape(src.Repo),
		escapeGithubSegments(ref), escapeGithubSegments(p))
}

func escapeGithubSegments(s string) string {
	segs := strings.Split(s, "/")
	for i, seg := range segs {
		segs[i] = url.PathEscape(seg)
	}
	return strings.Join(segs, "/")
}

type githubTreeResponse struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"tree"`
	Truncated bool `json:"truncated"`
}

func (imp *jailbreakImporter) defaultBranch(ctx context.Context, src githubJailbreakSource) (string, error) {
	var out struct {
		DefaultBranch string `json:"default_branch"`
	}
	u := fmt.Sprintf("%s/repos/%s/%s", imp.apiBase, url.PathEscape(src.Owner), url.PathEscape(src.Repo))
	if err := imp.getJSON(ctx, u, &out); err != nil {
		return "", err
	}
	if out.DefaultBranch == "" {
		return "", fmt.Errorf("repository %s/%s reports no default branch", src.Owner, src.Repo)
	}
	return out.DefaultBranch, nil
}

func (imp *jailbreakImporter) tree(ctx context.Context, src githubJailbreakSource, ref string) ([]githubTreeEntry, bool, error) {
	var out githubTreeResponse
	u := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1", imp.apiBase,
		url.PathEscape(src.Owner), url.PathEscape(src.Repo), escapeGithubSegments(ref))
	if err := imp.getJSON(ctx, u, &out); err != nil {
		return nil, false, err
	}
	entries := make([]githubTreeEntry, 0, len(out.Tree))
	for _, e := range out.Tree {
		if e.Type == "blob" {
			entries = append(entries, githubTreeEntry{Path: e.Path, Size: e.Size})
		}
	}
	return entries, out.Truncated, nil
}

func (imp *jailbreakImporter) getJSON(ctx context.Context, u string, out any) error {
	text, err := imp.fetchText(ctx, u, 4*1024*1024)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(text), out); err != nil {
		return fmt.Errorf("decode %s: %w", u, err)
	}
	return nil
}

// fetchText GETs u and returns at most limit bytes. A body past the limit is
// an error rather than a truncation: half a prompt would inject half a rule.
func (imp *jailbreakImporter) fetchText(ctx context.Context, u string, limit int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "9router-go")
	resp, err := imp.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("GET %s: status %d", u, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", u, err)
	}
	if int64(len(body)) > limit {
		return "", fmt.Errorf("GET %s: body exceeds %d bytes", u, limit)
	}
	return string(body), nil
}

// HandleImportJailbreakGithub imports a public GitHub repository of prompts as
// presets. POST /api/jailbreak/import-github with {"target":"owner/repo"}.
func (h *DashboardHandler) HandleImportJailbreakGithub(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024))
	if err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req struct {
		Target string `json:"target"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Target) == "" {
		handlerutil.WriteJSONError(w, http.StatusBadRequest, "missing target: want owner/repo or a github.com URL")
		return
	}
	res, err := h.importer().importPresets(r.Context(), h.Repo, req.Target)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errJailbreakBadTarget) {
			status = http.StatusBadRequest
		}
		handlerutil.WriteJSONError(w, status, err.Error())
		return
	}
	handlerutil.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"source":   res.Source,
		"imported": res.Imported,
		"refused":  res.Refused,
	})
}
