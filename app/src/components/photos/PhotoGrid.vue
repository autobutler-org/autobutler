<template>
  <div class="photo-grid">
    <PhotoGridItem
      v-for="photo in photos"
      :key="photo.relPath || photo.id"
      :photo="photo"
      @click="selectPhoto(photo)"
    />
  </div>
  <CirrusFileViewer
    v-model="fileViewerOpen"
    :file-path="selectedFilePath"
    :file-type="selectedFileType"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { FileType } from '@/types/cirrus'
import type { Photo } from '@/types/photo'
import PhotoGridItem from './PhotoGridItem.vue'
import CirrusFileViewer from '@/components/cirrus/CirrusFileViewer.vue'

defineProps<{
  photos: Photo[]
  page: number
  totalPhotos: number
}>()

const fileViewerOpen = ref(false)
const selectedFilePath = ref('')
const selectedFileType = ref<FileType>('image')

const selectPhoto = (photo: Photo) => {
  if (photo.relPath) {
    selectedFilePath.value = photo.relPath
    selectedFileType.value = 'image' // Assuming all photos are images; adjust as needed
    fileViewerOpen.value = true
  }
}
</script>

<style lang="scss" scoped>
.photo-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: $spacing-md;
  padding-bottom: $spacing-2xl;
}
</style>
