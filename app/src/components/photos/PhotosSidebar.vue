<template>
  <LibrarySidebar :sections="sections" />
</template>

<script setup lang="ts">
import LibrarySidebar from '@/components/common/LibrarySidebar.vue'

const props = defineProps<{ photoCount: number }>()

const sections = [
  {
    title: 'Library',
    items: [
      { label: 'All Photos', count: props.photoCount, active: true },
      { label: 'Albums', count: 0 },
      { label: 'Favorites', count: 0 },
      { label: 'Recently Added', count: 0 },
    ],
  },
  {
    title: 'Smart Albums',
    items: [
      { label: 'Screenshots', count: 0 },
      { label: 'Videos', count: 0 },
      { label: 'Selfies', count: 0 },
    ],
  },
]
</script>

<style lang="scss" scoped></style>
