<template>
  <div v-if="open" class="settings-modal-overlay" @click.self="close">
    <div class="settings-modal">
      <h2>Settings</h2>
      <div class="settings-section">
        <label
          >Font size scale:
          <input
            type="range"
            min="0.8"
            max="1.5"
            step="0.01"
            :value="settings.fontSizeScale"
            @input="
              settings.setFontSizeScale(
                Number(($event.target as HTMLInputElement).value),
              )
            "
          />
          <span>{{ settings.fontSizeScale.toFixed(2) }}x</span>
        </label>
      </div>
      <button class="close-btn" @click="close">Close</button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useSettingsStore } from '@/stores/settings'
import { watch } from 'vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits(['close'])
const settings = useSettingsStore()

const close = () => {
  emit('close')
}

// Focus trap and escape key
watch(
  () => props.open,
  (val) => {
    if (val) {
      setTimeout(() => {
        const el = document.querySelector('.settings-modal') as HTMLElement
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
.settings-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
}
.settings-modal {
  background: #222;
  color: #fff;
  min-width: 320px;
  max-width: 90vw;
  border-radius: 12px;
  box-shadow: 0 4px 32px #0008;
  padding: 2rem 2.5rem 1.5rem 2.5rem;
  outline: none;
}
.settings-section {
  margin-bottom: 1.5rem;
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
