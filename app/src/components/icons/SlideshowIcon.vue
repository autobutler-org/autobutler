<script setup lang="ts">
// Slideshow/presentation file icon
</script>

<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    class="icon icon--base icon--slideshow"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      stroke-linecap="round"
      stroke-linejoin="round"
      stroke-width="2"
      d="M8 13v-1m4 1v-3m4 3V8M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z"
    />
  </svg>
</template>

<style lang="scss" scoped>
.icon {
  display: inline-block;
  vertical-align: middle;

  &--base {
    width: 1.25rem;
    height: 1.25rem;
  }

  &--slideshow {
    color: var(--color-yellow-600);
    margin-right: var(--spacing-md);
  }
}
</style>
