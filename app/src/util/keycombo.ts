import { arrayEquals } from './array';

export type KeyModifier = 'alt' | 'ctrl' | 'meta' | 'shift';
export interface KeyCombo {
  modifiers?: KeyModifier[];
  keys: string[];
}

// Parses a key combination string (e.g., "ctrl-alt-delete") into a KeyCombo object
export const fromKeyComboString = (combo: string): KeyCombo => {
  const parts = combo
    .toLowerCase()
    .split('-')
    .filter((part) => part.length > 0);
  if (parts.length === 0) {
    return { keys: [] };
  }
  const modifiers = new Set<KeyModifier>();
  const keys = new Set<string>();

  for (const part of parts) {
    if (['alt', 'ctrl', 'meta', 'shift'].includes(part)) {
      modifiers.add(part as KeyModifier);
    } else {
      keys.add(part.toLowerCase());
    }
  }

  return {
    modifiers: modifiers.size > 0 ? Array.from(modifiers).sort() : [],
    keys: Array.from(keys).sort(),
  };
};

export const toKeyComboString = (combo: KeyCombo): string => {
  const parts: string[] = [];
  if (combo.modifiers) {
    parts.push(...combo.modifiers);
  }
  parts.push(...combo.keys);
  return parts.join('-');
};

export const areKeyCombosEqual = (a: KeyCombo, b: KeyCombo): boolean => {
  const aModifiers = a.modifiers ? [...a.modifiers].sort() : [];
  const bModifiers = b.modifiers ? [...b.modifiers].sort() : [];
  const aKeys = [...a.keys].sort();
  const bKeys = [...b.keys].sort();

  if (
    aModifiers.length !== bModifiers.length ||
    aKeys.length !== bKeys.length
  ) {
    return false;
  }

  return arrayEquals(aModifiers, bModifiers) && arrayEquals(aKeys, bKeys);
};

export const toEventListenerFunc =
  (combo: KeyCombo, handler: (e: Event) => boolean) => (e: Event) => {
    for (const mod of combo.modifiers ?? []) {
      switch (mod) {
        case 'alt':
          if (!(e as KeyboardEvent).altKey) {
            return false;
          }
          break;
        case 'ctrl':
          if (!(e as KeyboardEvent).ctrlKey) {
            return false;
          }
          break;
        case 'meta':
          if (!(e as KeyboardEvent).metaKey) {
            return false;
          }
          break;
        case 'shift':
          if (!(e as KeyboardEvent).shiftKey) {
            return false;
          }
          break;
      }
    }

    for (const key of combo.keys) {
      if (
        (e as KeyboardEvent).key.toLowerCase() === key.toLowerCase() ||
        (e as KeyboardEvent).code.toLowerCase() === key.toLowerCase()
      ) {
        return handler(e);
      }
    }

    return false;
  };
