<template>
  <svg class="sort-switcher-arrows" fill="currentColor" viewBox="0 0 24 24">
    <path d="M7 8l5-5 5 5z"></path>
    <path d="M7 16l5 5 5-5z"></path>
  </svg>
</template>

<style lang="scss" scoped>
.sort-switcher-folder {
  width: 1rem;
  height: 1rem;
  color: $color-gray-500;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-400;
  }
}
</style>
