<template>
  <div class="sort-switcher-icons">
    <svg class="sort-switcher-arrows" fill="currentColor" viewBox="0 0 24 24">
      <path d="M7 8l5-5 5 5z" />
      <path d="M7 16l5 5 5-5z" />
    </svg>
    <svg v-if="!mixedSorting" class="sort-switcher-folder" fill="currentColor" viewBox="0 0 20 20">
      <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
    </svg>
    <svg v-else class="sort-switcher-file" fill="currentColor" viewBox="0 0 20 20">
      <path
        fill-rule="evenodd"
        d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z"
        clip-rule="evenodd"
      />
    </svg>
  </div>
  <span class="sort-switcher-label">{{ mixedSorting ? 'Mixed' : 'Folders' }}</span>
</template>

<script setup lang="ts">
defineProps<{
  mixedSorting: boolean
}>()
</script>

<style lang="scss" scoped>
.sort-switcher-icons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  margin-bottom: $spacing-xs;
  height: 1rem;
  padding-left: $spacing-xs;
}

.sort-switcher-arrows,
.sort-switcher-folder,
.sort-switcher-file {
  width: 1rem;
  height: 1rem;
  color: $color-gray-500;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-400;
  }
}

.sort-switcher-label {
  color: $color-gray-500;
  font-weight: 600;
  width: 3rem;
  text-align: center;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-400;
  }
}
</style>
