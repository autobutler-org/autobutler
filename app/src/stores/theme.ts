import { defineStore } from 'pinia'

export const useThemeStore = defineStore('theme', {
  state: () => ({
    fontSizeBase: 1,
    fontSizeScale: 1, // Multiplier for all font sizes
  }),
  actions: {
    setFontSizeScale(scale: number) {
      console.log(`Setting font size scale to ${scale}`)
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
  },
})
