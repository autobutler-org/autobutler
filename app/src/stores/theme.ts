import { defineStore } from 'pinia'

import {
  getCssCustomProperties,
  keyToPropertyName,
  propertyNameToKey,
} from '@/util/style'

type Palette = Record<string, string>

export const useThemeStore = defineStore('theme', {
  state: () => {
    const styles = getCssCustomProperties().filter((prop) =>
      prop.name.startsWith('--theme-palette-'),
    )
    const theme = {
      fontSizeBase: 1,
      fontSizeScale: 1, // Multiplier for all font sizes
      palette: {} as Palette,
    }
    for (const style of styles) {
      const key = propertyNameToKey(style.name.replace('--theme-', ''))
      theme.palette[key] = style.value // Set defaults from CSS file, assuming store is uninitialized for now
    }
    return theme
  },
  actions: {
    setFontSizeScale(scale: number) {
      this.fontSizeScale = scale
      this.applyFontSizeScale()
    },
    applyFontSizeScale() {
      // These match the theme variables in variables.scss
      const baseSizes = {
        xs: 0.75,
        sm: 0.875,
        base: 1,
        lg: 1.125,
        xl: 1.25,
        '2xl': 1.5,
        '3xl': 1.875,
        '5xl': 3,
      }
      for (const [key, rem] of Object.entries(baseSizes)) {
        document.documentElement.style.setProperty(
          `--theme-font-size-${key}`,
          `${rem * this.fontSizeScale}rem`,
        )
      }
    },
    setPaletteColor(key: string, value: string) {
      this.palette[key] = value
      document.documentElement.style.setProperty(
        `--theme-palette-${keyToPropertyName(key, false)}`,
        value,
      )
    },
    applyPalette() {
      for (const [key, value] of Object.entries(this.palette)) {
        document.documentElement.style.setProperty(
          `--theme-palette-${keyToPropertyName(key, false)}`,
          value,
        )
      }
    },
  },
})
