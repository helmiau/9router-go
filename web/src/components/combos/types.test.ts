import { describe, expect, it } from 'bun:test'
import { COMBO_STRATEGIES, resolveComboStrategy, sortMembersByProvider } from './types'

const OPTION_VALUES = COMBO_STRATEGIES.map((s) => s.value)

describe('resolveComboStrategy', () => {
  // The server fills combo.strategy from settings.comboStrategies[name] and
  // otherwise from the global `comboStrategy`, so a card can receive either
  // vocabulary. Anything it can receive has to render as one of the options.
  it('maps every value the server can send onto an existing option', () => {
    const serverValues = [
      undefined,
      null,
      '',
      'fallback',
      'round-robin',
      'first-model',
      'sticky',
      'capacity',
      'fusion',
      'nonsense-from-a-hand-edited-backup'
    ]

    for (const value of serverValues) {
      expect(OPTION_VALUES).toContain(resolveComboStrategy(value))
    }
  })

  it('treats the global routing mode as the card fallback', () => {
    // `first-model` is the global Combo Routing Mode and the default when a
    // combo has no per-combo entry. It is try-in-order, which is exactly what
    // the card calls `fallback`.
    expect(resolveComboStrategy('first-model')).toBe('fallback')
  })

  it('passes a real per-combo strategy through unchanged', () => {
    for (const value of ['round-robin', 'sticky', 'capacity', 'fusion']) {
      expect(resolveComboStrategy(value)).toBe(value)
    }
  })

  it('falls back for a missing or unrecognised value', () => {
    expect(resolveComboStrategy(undefined)).toBe('fallback')
    expect(resolveComboStrategy(null)).toBe('fallback')
    expect(resolveComboStrategy('')).toBe('fallback')
    expect(resolveComboStrategy('nonsense')).toBe('fallback')
  })

  it('lists every option with a non-empty label', () => {
    for (const strategy of COMBO_STRATEGIES) {
      expect(strategy.value.length).toBeGreaterThan(0)
      expect(strategy.label.length).toBeGreaterThan(0)
    }
  })
})

describe('sortMembersByProvider', () => {
  // The tool is a one-shot grouping, not a standing sort: the operator presses
  // it and can drag anything back. So the order INSIDE a group must survive —
  // re-sorting within a provider would discard the fallback ordering the
  // operator arranged.
  it('groups by provider and keeps the order inside each group', () => {
    const input = ['ds/beta', 'oc/alpha', 'ds/alpha', 'cutad/gamma', 'oc/beta']
    expect(sortMembersByProvider(input)).toEqual([
      'ds/beta',
      'ds/alpha',
      'oc/alpha',
      'oc/beta',
      'cutad/gamma'
    ])
  })

  it('orders providers by first appearance, not alphabetically', () => {
    // The operator put zzz first on purpose; alphabetical provider order would
    // move it last and discard that.
    expect(sortMembersByProvider(['zzz/a', 'aaa/b'])).toEqual(['zzz/a', 'aaa/b'])
  })

  it('keeps members without a provider segment in a trailing group', () => {
    expect(sortMembersByProvider(['ds/a', 'bare', 'oc/b'])).toEqual(['ds/a', 'oc/b', 'bare'])
  })

  it('is a no-op on an already grouped list', () => {
    const grouped = ['ds/a', 'ds/b', 'oc/c']
    expect(sortMembersByProvider(grouped)).toEqual(grouped)
  })

  it('handles an empty list', () => {
    expect(sortMembersByProvider([])).toEqual([])
  })

  it('does not mutate the input', () => {
    const input = ['oc/a', 'ds/b']
    const copy = [...input]
    sortMembersByProvider(input)
    expect(input).toEqual(copy)
  })
})
