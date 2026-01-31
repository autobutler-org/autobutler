<template>
  <!-- TODO: Add back once the feature has more going on -->
  <!-- <LibrarySidebar :sections="sections" /> -->
  <template></template>
</template>

<script lang="ts" setup>
import { ref } from 'vue';

const props = defineProps<{ photoCount: number }>();

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const sections = ref([
  {
    title: 'Library (More coming soon)',
    items: [{ label: 'All Photos', count: props.photoCount, active: true }],
  },
]);
</script>
