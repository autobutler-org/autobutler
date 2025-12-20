<script setup lang="ts">
// Generic file icon
</script>

<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    class="icon icon--base icon--file"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      stroke-linecap="round"
      stroke-linejoin="round"
      stroke-width="2"
      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
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

  &--file {
    color: var(--color-gray-500);
    margin-right: var(--spacing-md);
  }
}
</style>
