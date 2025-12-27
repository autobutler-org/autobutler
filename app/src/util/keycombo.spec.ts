// @vitest-environment node
import { describe, it, expect, vi } from 'vitest'
import {
  fromKeyComboString,
  toKeyComboString,
  areKeyCombosEqual,
  toEventListenerFunc,
  type KeyCombo,
} from './keycombo'

describe('fromKeyComboString', () => {
  it('parses modifiers and keys', () => {
    expect(fromKeyComboString('ctrl-alt-delete')).toEqual({
      modifiers: ['alt', 'ctrl'],
      keys: ['delete'],
    })
    expect(fromKeyComboString('shift-a')).toEqual({
      modifiers: ['shift'],
      keys: ['a'],
    })
    expect(fromKeyComboString('meta-ctrl-x')).toEqual({
      modifiers: ['ctrl', 'meta'],
      keys: ['x'],
    })
  })

  it('handles no modifiers', () => {
    expect(fromKeyComboString('a')).toEqual({ modifiers: [], keys: ['a'] })
  })

  it('handles empty string', () => {
    expect(fromKeyComboString('')).toEqual({ keys: [] })
  })

  it('is case-insensitive and sorts', () => {
    expect(fromKeyComboString('CTRL-Alt-b')).toEqual({
      modifiers: ['alt', 'ctrl'],
      keys: ['b'],
    })
  })
})

describe('toKeyComboString', () => {
  it('joins modifiers and keys', () => {
    const combo: KeyCombo = { modifiers: ['ctrl', 'alt'], keys: ['delete'] }
    expect(toKeyComboString(combo)).toBe('ctrl-alt-delete')
  })
  it('handles no modifiers', () => {
    const combo: KeyCombo = { keys: ['a'] }
    expect(toKeyComboString(combo)).toBe('a')
  })
})

describe('areKeyCombosEqual', () => {
  it('returns true for equal combos', () => {
    const a: KeyCombo = { modifiers: ['ctrl'], keys: ['a'] }
    const b: KeyCombo = { modifiers: ['ctrl'], keys: ['a'] }
    expect(areKeyCombosEqual(a, b)).toBe(true)
  })
  it('returns false for different combos', () => {
    const a: KeyCombo = { modifiers: ['ctrl'], keys: ['a'] }
    const b: KeyCombo = { modifiers: ['alt'], keys: ['a'] }
    expect(areKeyCombosEqual(a, b)).toBe(false)
  })
  it('ignores order of modifiers and keys', () => {
    const a: KeyCombo = { modifiers: ['ctrl', 'alt'], keys: ['a', 'b'] }
    const b: KeyCombo = { modifiers: ['alt', 'ctrl'], keys: ['b', 'a'] }
    expect(areKeyCombosEqual(a, b)).toBe(true)
  })
})

describe('toEventListenerFunc', () => {
  it('calls handler if combo matches event', () => {
    const combo: KeyCombo = { modifiers: ['ctrl'], keys: ['a'] }
    const handler = vi.fn().mockReturnValue(true)
    const listener = toEventListenerFunc(combo, handler)
    const event = { ctrlKey: true, key: 'a', code: 'KeyA' } as KeyboardEvent
    expect(listener(event)).toBe(true)
    expect(handler).toHaveBeenCalledWith(event)
  })
  it('returns false if modifiers do not match', () => {
    const combo: KeyCombo = { modifiers: ['alt'], keys: ['a'] }
    const handler = vi.fn()
    const listener = toEventListenerFunc(combo, handler)
    const event = { ctrlKey: true, key: 'a', code: 'KeyA' } as KeyboardEvent
    expect(listener(event)).toBe(false)
    expect(handler).not.toHaveBeenCalled()
  })
  it('returns false if key does not match', () => {
    const combo: KeyCombo = { modifiers: ['ctrl'], keys: ['b'] }
    const handler = vi.fn()
    const listener = toEventListenerFunc(combo, handler)
    const event = { ctrlKey: true, key: 'a', code: 'KeyA' } as KeyboardEvent
    expect(listener(event)).toBe(false)
    expect(handler).not.toHaveBeenCalled()
  })
})
