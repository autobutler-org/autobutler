<template>
  <LibrarySidebar :sections="sections" />
</template>

<script lang="ts" setup>
import LibrarySidebar from '@/components/common/LibrarySidebar.vue';
import { ref } from 'vue';

const props = defineProps<{ photoCount: number }>();

const sections = ref([
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
]);
</script>
