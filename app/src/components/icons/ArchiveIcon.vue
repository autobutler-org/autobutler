<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    class="icon icon--base icon--archive"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      stroke-linecap="round"
      stroke-linejoin="round"
      stroke-width="2"
      d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
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

  &--archive {
    color: $color-green-600;
    margin-right: $spacing-md;
  }
}
</style>
