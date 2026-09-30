/**
 * Per-group windowing for the model picker.
 *
 * The picker renders every model of every provider. With ~36 groups and ~890
 * pills, returning to the full list after clearing a search costs 111-131ms
 * (measured), and scrolling has no JS handler at all - the browser just paints
 * 890 nodes.
 *
 * An earlier attempt windowed individual rows and assumed a fixed 30px each.
 * That broke the layout: pills are laid out with `flex flex-wrap`, so a group
 * can occupy three rows and the next one a single row. Guessing heights made
 * the list jump.
 *
 * So the unit here is a whole provider group. A group renders as one intact
 * block, exactly as today; only which groups are mounted changes. Heights are
 * measured with a ResizeObserver and unmeasured ones fall back to a running
 * average, so the scroll height stays put instead of jumping as measurements
 * arrive. A searched list is left fully rendered - the results are small
 * (142 pills, ~13ms) and the user needs to see all of them.
 */

/** Groups rendered beyond the viewport, to hide scroll tearing. */
export const GROUP_OVERSCAN = 2

export interface GroupHeights {
  /** Real height per group id, filled in by the ResizeObserver. */
  measured: Record<string, number>
  /** Fallback height for groups that have not been rendered (or measured) yet. */
  estimated: number
}

export interface GroupWindowState {
  heights: GroupHeights
}

export function createGroupWindow(estimated = 180): GroupWindowState {
  return { heights: { measured: {}, estimated } }
}

/** Record a group's real height. Returns false when nothing changed. */
export function measureGroup(state: GroupWindowState, id: string, height: number): boolean {
  if (!Number.isFinite(height) || height <= 0) return false
  if (state.heights.measured[id] === height) return false
  state.heights.measured[id] = height
  return true
}

/** Average of the measured heights, or the estimate when nothing is known yet. */
export function averageHeight(h: GroupHeights): number {
  const values = Object.values(h.measured)
  if (values.length === 0) return h.estimated
  let sum = 0
  for (const v of values) sum += v
  return sum / values.length
}

/** Offset of every group, plus the total height. */
export function groupOffsets(
  h: GroupHeights,
  ids: string[]
): { offsets: number[]; total: number } {
  const avg = averageHeight(h)
  const offsets: number[] = []
  let y = 0
  for (const id of ids) {
    offsets.push(y)
    y += h.measured[id] ?? avg
  }
  return { offsets, total: y }
}

export interface GroupWindow {
  start: number
  end: number
  topSpacer: number
  bottomSpacer: number
  totalHeight: number
}

/**
 * Which groups to mount for a scroll position. Group indices are contiguous, so
 * the rendered list keeps the same shape as the full one.
 */
export function groupWindow(
  h: GroupHeights,
  ids: string[],
  scrollTop: number,
  viewportHeight: number,
  overscan = GROUP_OVERSCAN
): GroupWindow {
  const total = ids.length
  if (total === 0) return { start: 0, end: 0, topSpacer: 0, bottomSpacer: 0, totalHeight: 0 }

  const { offsets, total: totalHeight } = groupOffsets(h, ids)

  // Unknown viewport (first paint) or a list that fits: render the lot, exactly
  // as before. A wrong guess here is what disabled the first attempt.
  if (viewportHeight <= 0 || totalHeight <= viewportHeight) {
    return { start: 0, end: total, topSpacer: 0, bottomSpacer: 0, totalHeight }
  }

  let lo = 0
  let hi = total
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (offsets[mid] <= scrollTop) lo = mid + 1
    else hi = mid
  }
  const firstVisible = Math.max(0, lo - 1)

  const start = Math.max(0, firstVisible - overscan)

  let end = firstVisible
  while (end < total && offsets[end] < scrollTop + viewportHeight) end++
  end = Math.min(total, end + overscan)

  // Clamp before indexing: a deep scroll can leave `end` at `total`, and
  // offsets only has `total` entries, so offsets[end] would be undefined.
  const endIndex = Math.min(end, total)
  return {
    start,
    end: endIndex,
    topSpacer: offsets[start],
    bottomSpacer: totalHeight - (endIndex < total ? offsets[endIndex] : totalHeight),
    totalHeight,
  }
}
