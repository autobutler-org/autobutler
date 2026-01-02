<template>
  <div class="photo-grid-item">
    <img
      class="photo-grid-image"
      :src="thumbnailPath"
      :alt="photo.fileName || ''"
      loading="lazy"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';

const props = defineProps<{
  photo: { relPath?: string; fileName?: string; id?: string };
}>();

const thumbnailPath = computed(() => {
  // Use relPath or id to build the thumbnail URL
  if (props.photo.relPath) {
    return `/api/v1/thumbnails/${props.photo.relPath}`;
  }
  return '';
});
</script>

<style lang="scss" scoped>
.photo-grid-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.photo-grid-item {
  position: relative;
  aspect-ratio: 1;
  border-radius: $border-radius-lg;
  overflow: hidden;
  cursor: pointer;
  background: $theme-palette-bg-secondary;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease;

  &:hover {
    transform: scale(1.02);
    box-shadow: $shadow-lg;
    z-index: 1;
  }
}
</style>
