<template>
  <div class="modal-overlay" @click.self="onClose">
    <button class="modal-close" @click="onClose" aria-label="Close">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        fill="none"
        viewBox="0 0 20 20"
      >
        <path
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          d="M5 5l10 10M15 5l-10 10"
        />
      </svg>
    </button>
    <div class="modal-content">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
const emit = defineEmits(['close'])
function onClose() {
  emit('close')
}
</script>

<style lang="scss" scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
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
}
.modal-close:hover {
  background: #f2f2f2;
}
.modal-close svg {
  display: block;
  margin: auto;
}
</style>
