<template>
  <div class="landing-body">
    <TopNav :navLinks="navLinks" />
    <div class="photos-library">
      <div class="photos-container">
        <PhotosSidebar :photo-count="totalPhotos" :summary="summary" />
        <div id="mobile-photos-arrival-location"></div>
        <div class="photos-main">
          <div class="photos-header">
            <h2 class="photos-title" @click="scrollToArrival">All Photos</h2>
            <div class="photos-count">{{ formatPhotoCount(totalPhotos) }}</div>
          </div>
          <div class="photos-grid-container">
            <PhotoGrid :photos="photos" :page="1" :total-photos="totalPhotos" @photoClick="onPhotoClick" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import PhotosSidebar from '@/components/photos/PhotosSidebar.vue'
import PhotoGrid from '@/components/photos/PhotoGrid.vue'
import { fetchPhotos, type PhotoApiResponse } from '@/services/photosService'
import TopNav from '@/components/home/TopNav.vue'
import type { NavLink } from '@/types/home'

const navLinks: NavLink[] = [
  { name: 'Cirrus', href: '/cirrus' },
  { name: 'Photos', href: '/photos' },
  { name: 'Books', href: '/books' },
]

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
.photos-library {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100vw;
}
.photos-container {
  display: flex;
  height: 100vh;
  max-height: 100vh;
  max-width: 100vw;
  overflow: hidden;
}
.photos-main {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  margin: var(--spacing-xl) 0;
  background: transparent;
}
.photos-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-lg) 0;
  border-bottom: 1px solid var(--color-gray-200);
}
.photos-title {
  font-size: var(--font-size-3xl);
  font-weight: 700;
  color: var(--color-gray-900);
  margin: 0;
}

@media (prefers-color-scheme: dark) {
  .photos-title {
    color: white;
  }
}
.photos-count {
  font-size: var(--font-size-lg);
  color: var(--color-gray-500);
}
.photos-grid-container {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
  background: transparent;
  padding: 0;
}
</style>
