<template>
  <svg class="sort-arrow sort-arrow--active" fill="currentColor" viewBox="0 0 24 24">
    <path d="M7 14l5-5 5 5z"></path>
  </svg>
</template>

<style lang="scss" scoped>
.sort-arrow {
  width: 16px;
  height: 16px;
  color: $color-gray-400;

  &--active {
    color: $color-gray-700;

    @media (prefers-color-scheme: dark) {
      color: $color-gray-300;
    }
  }
}
</style>
