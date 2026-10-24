package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
)

type H struct{}

func (h *H) HandleGetJailbreak(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(`{"id":"default"}`))
}
func (h *H) HandlePutJailbreak(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
func (h *H) HandleListJailbreakPresets(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(`[]`))
}
func (h *H) HandlePutJailbreakPreset(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
func (h *H) HandleDeleteJailbreakPreset(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
func (h *H) HandleListJailbreakModels(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(`{}`))
}
func (h *H) HandleSetJailbreakModel(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
func (h *H) HandleGetJailbreakExcludes(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(`[]`))
}
func (h *H) HandlePutJailbreakExcludes(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
func (h *H) HandleImportJailbreak(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
func (h *H) HandleUploadJailbreakPreset(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }

func main() {
	r := chi.NewRouter()
	h := &H{}

	r.Get("/api/jailbreak", h.HandleGetJailbreak)
	r.Put("/api/jailbreak", h.HandlePutJailbreak)
	r.Get("/api/jailbreak/presets", h.HandleListJailbreakPresets)
	r.Put("/api/jailbreak/presets", h.HandlePutJailbreakPreset)
	r.Delete("/api/jailbreak/presets/{id}", h.HandleDeleteJailbreakPreset)
	r.Get("/api/jailbreak/models", h.HandleListJailbreakModels)
	r.Put("/api/jailbreak/models", h.HandleSetJailbreakModel)
	r.Get("/api/jailbreak/excludes", h.HandleGetJailbreakExcludes)
	r.Put("/api/jailbreak/excludes", h.HandlePutJailbreakExcludes)
	r.Post("/api/jailbreak/import", h.HandleImportJailbreak)
	r.Post("/api/jailbreak/presets/upload", h.HandleUploadJailbreakPreset)

	for _, ep := range []string{
		"/api/jailbreak",
		"/api/jailbreak/presets",
		"/api/jailbreak/models",
		"/api/jailbreak/excludes",
		"/api/jailbreak/import",
		"/api/jailbreak/presets/upload",
	} {
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+ep, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		fmt.Printf("GET %s -> %d\n", ep, rec.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "http://localhost/api/jailbreak/import", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	fmt.Printf("POST /api/jailbreak/import -> %d\n", rec.Code)
}
