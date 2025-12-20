<template>
  <div class="photo-grid">
    <PhotoGridItem v-for="photo in photos" :key="photo.relPath || photo.id" :photo="photo" @click="() => $emit('photoClick', photo)" />
    <!-- TODO: Infinite scroll loader -->
  </div>
</template>

<script setup lang="ts">
import PhotoGridItem from './PhotoGridItem.vue'
import { defineProps } from 'vue'

defineProps<{
  photos: any[]
  page: number
  totalPhotos: number
}>()

defineEmits(['photoClick'])
</script>

<style lang="scss" scoped>
.photo-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--spacing-md);
  padding-bottom: var(--spacing-2xl);
}
</style>
