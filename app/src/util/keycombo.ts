import { arrayEquals } from "./array"

export type KeyModifier = 'alt' | 'ctrl' | 'meta' | 'shift';
export interface KeyCombo {
  modifiers?: KeyModifier[];
  keys: string[];
}

// Parses a key combination string (e.g., "ctrl-alt-delete") into a KeyCombo object
export const parseKeyCombo = (combo: string): KeyCombo => {
  const parts = combo.toLowerCase().split('-')
  if (parts.length === 0) {
    return { keys: [] }
  }
  const modifiers = new Set<KeyModifier>()
  const keys = new Set<string>()

  for (const part of parts) {
    if (['alt', 'ctrl', 'meta', 'shift'].includes(part)) {
      modifiers.add(part as KeyModifier)
    } else {
      keys.add(part.toLowerCase())
    }
  }

  return {
    modifiers: modifiers.size > 0 ? Array.from(modifiers).sort() : [],
    keys: Array.from(keys).sort(),
  }
}

export const toKeyComboString = (combo: KeyCombo): string => {
  const parts: string[] = []
  if (combo.modifiers) {
    parts.push(...combo.modifiers)
  }
  parts.push(...combo.keys)
  return parts.join('-')
}

export const areKeyCombosEqual = (a: KeyCombo, b: KeyCombo): boolean => {
  const aModifiers = a.modifiers ? [...a.modifiers].sort() : []
  const bModifiers = b.modifiers ? [...b.modifiers].sort() : []
  const aKeys = [...a.keys].sort()
  const bKeys = [...b.keys].sort()

  if (aModifiers.length !== bModifiers.length || aKeys.length !== bKeys.length) {
    return false
  }

  return arrayEquals(
    aModifiers,
    bModifiers
  ) && arrayEquals(
    aKeys,
    bKeys
  )
}
