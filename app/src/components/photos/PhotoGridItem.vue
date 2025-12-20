<template>
  <div class="photo-grid-item" @click="$emit('click')">
    <img class="photo-grid-image" :src="thumbnailPath" :alt="photo.fileName || ''" loading="lazy" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ photo: { relPath?: string; fileName?: string; id?: string } }>()

const thumbnailPath = computed(() => {
  // Use relPath or id to build the thumbnail URL
  if (props.photo.relPath) {
    return `/api/v1/thumbnails/${props.photo.relPath}`
  }
  return ''
})
</script>

<style lang="scss" scoped>
.photo-grid-item {
  position: relative;
  aspect-ratio: 1;
  border-radius: var(--border-radius-lg);
  overflow: hidden;
  cursor: pointer;
  background: var(--color-gray-200);
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease;
}
.photo-grid-item:hover {
  transform: scale(1.02);
  box-shadow: var(--shadow-lg);
  z-index: 1;
}
.photo-grid-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
</style>
