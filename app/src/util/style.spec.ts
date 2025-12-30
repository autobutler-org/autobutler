import { describe, expect, it } from 'vitest'
import { keyToPropertyName, propertyNameToKey } from './style'

describe('propertyNameToKey', () => {
  it('converts CSS custom property to camelCase key', () => {
    expect(propertyNameToKey('--theme-palette-accent')).toBe(
      'themePaletteAccent',
    )
    expect(propertyNameToKey('--foo-bar-baz')).toBe('fooBarBaz')
    expect(propertyNameToKey('--a')).toBe('a')
    expect(propertyNameToKey('--foo')).toBe('foo')
  })
})

describe('keyToPropertyName', () => {
  it('converts camelCase key to CSS custom property', () => {
    expect(keyToPropertyName('themePaletteAccent')).toBe(
      '--theme-palette-accent',
    )
    expect(keyToPropertyName('fooBarBaz')).toBe('--foo-bar-baz')
    expect(keyToPropertyName('a')).toBe('--a')
    expect(keyToPropertyName('foo')).toBe('--foo')
  })
})
