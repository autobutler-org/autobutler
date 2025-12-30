<template>
  <ModalDialog
    v-if="open"
    @close="close"
    :backdrop="false"
    :hide-close-button="true"
    :transparent="true"
  >
    <div class="theme-modal">
      <button class="theme-modal-close" @click="close" aria-label="Close">
        <CloseIcon />
      </button>
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
        <label class="palette-bg-primary">
          Primary Background Color:
          <input
            type="color"
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
    </div>
  </ModalDialog>
</template>

<script lang="ts" setup>
import ModalDialog from '@/components/common/ModalDialog.vue'
import CloseIcon from '@/components/icons/CloseIcon.vue'
import { useThemeStore } from '@/stores/theme'

defineProps<{ open: boolean }>()

const emit = defineEmits(['close'])
const theme = useThemeStore()
const close = () => emit('close')
</script>

<style lang="scss" scoped>
.theme-modal {
  position: relative;
  background: $theme-palette-bg-nav;
  color: $theme-palette-text-primary;
  min-width: 320px;
  max-width: 90vw;
  border-radius: 12px;
  box-shadow: 0 4px 32px rgba($theme-palette-bg-primary, 0.53);
  padding: 2rem 2.5rem 1.5rem 2.5rem;
  outline: none;
  h2 {
    margin-bottom: 1.2em;
  }
}

.theme-modal-close {
  position: absolute;
  top: 1.5rem;
  right: 1.5rem;
  background: $theme-palette-bg-inverse;
  border: none;
  border-radius: 50%;
  color: $theme-palette-text-inverse;
  cursor: pointer;
  z-index: 1100;
  box-shadow: 0 2px 8px rgba($theme-palette-bg-primary, 0.1);
  width: 2.5rem;
  height: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
  padding: 0;

  svg {
    display: block;
    margin: auto;
  }

  &:hover {
    background: $theme-palette-accent-hover;
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
</style>
