<template>
  <div
    :class="[backdrop ? 'modal-overlay' : 'modal-overlay--no-backdrop']"
    @click.self="onClose"
  >
    <button
      v-if="!hideCloseButton"
      class="modal-close"
      @click="onClose"
      aria-label="Close"
    >
      <CloseIcon />
    </button>
    <div
      :class="[transparent ? 'modal-content--transparent' : 'modal-content']"
    >
      <slot />
    </div>
  </div>
</template>

<script lang="ts" setup>
import CloseIcon from '@/components/icons/CloseIcon.vue'

const {
  backdrop = true,
  hideCloseButton = false,
  transparent = false,
} = defineProps<{
  backdrop?: boolean
  hideCloseButton?: boolean
  transparent?: boolean
}>()
const emit = defineEmits(['close'])
const onClose = () => {
  emit('close')
}
</script>

<style lang="scss" scoped>
.modal-close {
  position: fixed;
  top: 1.5rem;
  right: 1.5rem;
  background: white;
  border: none;
  border-radius: 50%;
  color: #222;
  cursor: pointer;
  z-index: 1100;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
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
    background: #f2f2f2;
  }
}

.modal-content {
  background: white;
}
.modal-content--transparent {
  background: transparent;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
}
.modal-overlay--no-backdrop {
  position: fixed;
  inset: 0;
  background: transparent;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
