import { defineStore } from 'pinia'

export const useSettingsStore = defineStore('settings', {
  state: () => ({
    fontSizeBase: 1,
    fontSizeScale: 1, // Multiplier for all font sizes
  }),
  actions: {
    setFontSizeScale(scale: number) {
      this.fontSizeScale = scale
      this.applyFontSizeScale()
    },
    applyFontSizeScale() {
      // These match the CSS variable names in variables.scss and App.vue
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
          `--font-size-${key}`,
          `${rem * this.fontSizeScale}rem`
        )
      }
    },
  },
})
