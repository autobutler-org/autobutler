<template>
  <div v-if="open" class="theme-modal-overlay" @click.self="close">
    <div class="theme-modal">
      <h2>Theme</h2>
      <div class="theme-section">
        <label class="font-size-label">
          Font size scale:
          <input
            type="range"
            min="0.5"
            max="1.5"
            step="0.01"
            :value="theme.fontSizeScale"
            @input="
              theme.setFontSizeScale(
                Number(($event.target as HTMLInputElement).value),
              )
            "
          />
          <span class="slider-value"
            >{{ theme.fontSizeScale.toFixed(2) }}x</span
          >
        </label>
      </div>
      <button class="close-btn" @click="close">Close</button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useThemeStore } from '@/stores/theme'
import { watch } from 'vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits(['close'])
const theme = useThemeStore()

const close = () => {
  emit('close')
}

// Focus trap and escape key
watch(
  () => props.open,
  (val) => {
    if (val) {
      setTimeout(() => {
        const el = document.querySelector('.theme-modal') as HTMLElement
        if (el) el.focus()
      }, 10)
      const handler = (e: KeyboardEvent) => {
        if (e.key === 'Escape') close()
      }
      window.addEventListener('keydown', handler)
      return () => window.removeEventListener('keydown', handler)
    }
  },
)
</script>

<style lang="scss" scoped>
.theme-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
}
.theme-modal {
  background: #222;
  color: #fff;
  min-width: 320px;
  max-width: 90vw;
  border-radius: 12px;
  box-shadow: 0 4px 32px #0008;
  padding: 2rem 2.5rem 1.5rem 2.5rem;
  outline: none;
  h2 {
    margin-bottom: 1.2em;
  }
}
.theme-section {
  margin-bottom: 1.5rem;
}
.font-size-label {
  display: flex;
  align-items: center;
  gap: 0.75em;
  font-size: 1.1em;
}
.font-size-label input[type='range'] {
  margin-left: 0.5em;
  margin-right: 0.5em;
}
.slider-value {
  min-width: 3em;
  text-align: right;
  display: inline-block;
}
.close-btn {
  background: #444;
  color: #fff;
  border: none;
  border-radius: 6px;
  padding: 0.5rem 1.2rem;
  font-size: 1rem;
  cursor: pointer;
  margin-top: 0.5rem;
}
.close-btn:hover {
  background: #666;
}
</style>
