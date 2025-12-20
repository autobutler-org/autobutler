<template>
  <LibraryLayout>
    <template #sidebar>
      <PhotosSidebar :photo-count="totalPhotos" :summary="summary" />
      <div id="mobile-photos-arrival-location"></div>
    </template>
    <template #title>
        <h2 class="library-title" @click="scrollToArrival">All Photos</h2>
    </template>
    <template #subtitle>
      <div class="library-subtitle">{{ formatPhotoCount(totalPhotos) }}</div>
    </template>
    <template #main>
      <div class="photos-grid-container">
        <PhotoGrid
          :photos="photos"
          :page="1"
          :total-photos="totalPhotos"
          @photoClick="onPhotoClick"
        />
      </div>
    </template>
  </LibraryLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import PhotosSidebar from '@/components/photos/PhotosSidebar.vue'
import PhotoGrid from '@/components/photos/PhotoGrid.vue'
import { fetchPhotos, type PhotoApiResponse } from '@/services/photosService'
import LibraryLayout from '@/components/common/LibraryLayout.vue'

const photos = ref<Photo[]>([])
const totalPhotos = ref(0)
const summary = ref({})
import { useRouter } from 'vue-router'
import type { Photo } from '@/types/photo'
const router = useRouter()

const onPhotoClick = (photo: Photo) => {
  if (photo.relPath) {
    router.push({ name: 'photo-viewer', params: { path: encodeURIComponent(photo.relPath) } })
  }
}

const formatPhotoCount = (count: number) => {
  if (count === 1) return '1 photo'
  return `${count.toLocaleString()} photos`
}

const scrollToArrival = () => {
  const arrival = document.getElementById('mobile-photos-arrival-location')
  if (arrival) {
    arrival.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

const convertPhotoApi = (photo: PhotoApiResponse) => {
  return {
    relPath: photo.relPath,
    fileName: photo.fileName,
    id: photo.relPath,
    size: photo.size,
    mtime: photo.mtime,
  }
}

onMounted(async () => {
  try {
    const photoList = await fetchPhotos()
    photos.value = photoList.map(convertPhotoApi)
    totalPhotos.value = photoList.length
  } catch (e) {
    // TODO: handle error
    console.error(e)
  }
})
</script>

<style lang="scss" scoped>
.photos-grid-container {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
  padding: 0;
}
</style>
