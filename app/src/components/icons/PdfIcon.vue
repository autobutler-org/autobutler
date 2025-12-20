<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    class="icon icon--base icon--pdf"
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      stroke-linecap="round"
      stroke-linejoin="round"
      stroke-width="2"
      d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
    />
    <text x="7" y="16" font-size="6" fill="currentColor" stroke="none">PDF</text>
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

  &--pdf {
    color: var(--color-red-600);
    margin-right: var(--spacing-md);
  }
}
</style>
