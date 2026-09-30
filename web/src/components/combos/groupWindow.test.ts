import assert from 'node:assert'
import { describe, it } from 'node:test'
import { createGroupWindow, measureGroup, groupWindow, averageHeight, type GroupHeights } from './groupWindow'

const VP = 300

const ids = (n: number, prefix = 'g') => Array.from({ length: n }, (_, i) => `${prefix}${i}`)

describe('groupWindow', () => {
  it('renders every group when they all fit the viewport', () => {
    const h: GroupHeights = { measured: { a: 100, b: 100, c: 50 }, estimated: 80 }
    const w = groupWindow(h, ['a', 'b', 'c'], 0, VP)
    assert.equal(w.start, 0)
    assert.equal(w.end, 3)
    assert.equal(w.topSpacer, 0)
    assert.equal(w.bottomSpacer, 0)
  })

  it('renders everything when the viewport is not measured yet', () => {
    const h: GroupHeights = { measured: {}, estimated: 200 }
    const w = groupWindow(h, ids(50), 0, 0)
    assert.equal(w.end, 50)
  })

  it('handles an empty group list', () => {
    const h: GroupHeights = { measured: {}, estimated: 200 }
    const w = groupWindow(h, [], 0, VP)
    assert.equal(w.totalHeight, 0)
    assert.equal(w.end, 0)
  })

  it('renders only a handful of groups on a long list', () => {
    const h: GroupHeights = { measured: {}, estimated: 200 }
    const w = groupWindow(h, ids(100), 0, VP)
    const rendered = w.end - w.start
    assert.ok(rendered <= 8, `rendered ${rendered} groups, expected a small window`)
  })

  it('keeps the group range contiguous, so the list shape is unchanged', () => {
    const h: GroupHeights = { measured: {}, estimated: 200 }
    const w = groupWindow(h, ids(100), 0, VP)
    assert.ok(Number.isInteger(w.start) && Number.isInteger(w.end))
    assert.ok(w.end > w.start)
    assert.ok(w.start >= 0 && w.end <= 100)
  })

  it('moves the window down as the user scrolls', () => {
    const h: GroupHeights = { measured: {}, estimated: 200 }
    const top = groupWindow(h, ids(100), 0, VP)
    const down = groupWindow(h, ids(100), 5000, VP)
    assert.ok(down.start > top.start)
  })

  it('never runs past the end of the list', () => {
    const h: GroupHeights = { measured: {}, estimated: 200 }
    const w = groupWindow(h, ids(10), 999999, VP)
    assert.equal(w.end, 10)
    assert.equal(w.bottomSpacer, 0)
  })

  it('never starts before the first group', () => {
    const h: GroupHeights = { measured: {}, estimated: 200 }
    const w = groupWindow(h, ids(10), -500, VP)
    assert.equal(w.start, 0)
  })

  it('handles a group taller than the whole viewport', () => {
    const h: GroupHeights = { measured: { a: 1000 }, estimated: 100 }
    const w = groupWindow(h, ['a', 'b', 'c', 'd'], 0, VP)
    assert.ok(w.end > 1, 'a group taller than the viewport still needs neighbours')
    assert.ok(w.totalHeight >= 1000)
  })

  it('never lets the spacers exceed the total height', () => {
    const h: GroupHeights = { measured: { a: 150, b: 90 }, estimated: 200 }
    const w = groupWindow(h, ids(60), 3000, VP)
    assert.ok(w.topSpacer >= 0, 'top spacer must not be negative')
    assert.ok(w.bottomSpacer >= 0, 'bottom spacer must not be negative')
    assert.ok(w.topSpacer + w.bottomSpacer <= w.totalHeight)
  })

  it('spacers plus the rendered groups account for the full height', () => {
    // With uniform heights the arithmetic is exact: whatever is not mounted is
    // covered by the two spacers, so the scrollbar never shifts.
    const h: GroupHeights = { measured: {}, estimated: 200 }
    const w = groupWindow(h, ids(60), 3000, VP)
    assert.equal(w.topSpacer + w.bottomSpacer + (w.end - w.start) * 200, w.totalHeight)
  })

  it('keeps the total height stable as measurements arrive', () => {
    const before = groupWindow({ measured: {}, estimated: 200 }, ids(50), 0, VP)
    const after = groupWindow({ measured: { g0: 320, g1: 80 }, estimated: 200 }, ids(50), 0, VP)
    // 320 and 80 average to exactly the 200px estimate, so the total must not
    // move - otherwise the list would jump while scrolling.
    assert.equal(before.totalHeight, after.totalHeight)
  })

  it('grows the total height when a measured group is bigger than the estimate', () => {
    const before = groupWindow({ measured: {}, estimated: 100 }, ids(10), 0, VP)
    const after = groupWindow({ measured: { g0: 900 }, estimated: 100 }, ids(10), 0, VP)
    assert.ok(after.totalHeight > before.totalHeight)
  })

  it('keeps the group under the scroll position inside the window', () => {
    // The whole point of windowing: whatever the user scrolled to must be
    // mounted, or the scroll would land on empty space.
    const h: GroupHeights = { measured: {}, estimated: 200 }
    const list = ids(100)
    for (const at of [0, 700, 2500, 8000, 19800]) {
      const w = groupWindow(h, list, at, VP)
      assert.ok(w.start <= at / 200 && (at / 200) < w.end, `scrollTop ${at} fell outside [${w.start}, ${w.end})`)
    }
  })
})

describe('averageHeight', () => {
  it('falls back to the estimate when nothing is measured', () => {
    assert.equal(averageHeight({ measured: {}, estimated: 175 }), 175)
  })

  it('averages what has been measured', () => {
    assert.equal(averageHeight({ measured: { a: 100, b: 300 }, estimated: 175 }), 200)
  })
})

describe('measureGroup', () => {
  it('records a group height and reports the change', () => {
    const st = createGroupWindow()
    assert.equal(measureGroup(st, 'a', 240), true)
    assert.equal(st.heights.measured.a, 240)
  })

  it('ignores an unchanged height, so no re-render is triggered', () => {
    const st = createGroupWindow()
    measureGroup(st, 'a', 240)
    assert.equal(measureGroup(st, 'a', 240), false)
  })

  it('ignores a nonsensical height', () => {
    const st = createGroupWindow()
    assert.equal(measureGroup(st, 'a', 0), false)
    assert.equal(measureGroup(st, 'a', -5), false)
    assert.equal(measureGroup(st, 'a', Number.NaN), false)
  })
})
