<template>
  <LibrarySidebar :sections="sections" />
</template>

<script setup lang="ts">
import LibrarySidebar from '@/components/common/LibrarySidebar.vue'

const sections = [
  {
    title: 'Library',
    items: [
      { label: 'All Books', active: true },
      { label: 'Shelves' },
      { label: 'Favorites' },
      { label: 'Recently Added' },
    ],
  },
  {
    title: 'Smart Shelves',
    items: [
      { label: 'PDFs' },
      { label: 'EPUBs' },
    ],
  },
]
</script>

<style scoped>
</style>
