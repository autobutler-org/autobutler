<template>
  <div class="modal-overlay" @click.self="onClose">
    <button class="modal-close" @click="onClose" aria-label="Close">
      <CloseIcon />
    </button>
    <div class="modal-content">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import CloseIcon from '@/components/icons/CloseIcon.vue'

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
  border-radius: 8px;
  box-shadow: 0 2px 16px rgba(0, 0, 0, 0.18);
  padding: 0;
  max-width: 95vw;
  max-height: 95vh;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
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
</style>
