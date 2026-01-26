<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    class="download-icon"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <!-- Download icon -->
    <path
      stroke-linecap="round"
      stroke-linejoin="round"
      stroke-width="2"
      d="M4 16v2a2 2 0 002 2h12a2 2 0 002-2v-2m-4-4l-4 4m0 0l-4-4m4 4V4"
    />
  </svg>
</template>
<style lang="scss" scoped>
.download-icon {
  width: 32px;
  height: 32px;
  color: $color-gray-500;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-400;
  }
}
</style>
