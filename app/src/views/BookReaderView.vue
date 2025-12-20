<template>
  <TopNav :navLinks="navLinks" />
  <div class="book-reader-view">
    <div class="book-reader-nav">
      <button class="book-reader-btn" @click="$router.back()" title="Back to library">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M19 12H5m0 0l7 7m-7-7l7-7"></path>
        </svg>
        <span>Library</span>
      </button>
      <div class="book-reader-info">
        <span class="book-reader-title">{{ fileName }}</span>
      </div>
      <div class="book-reader-spacer"></div>
    </div>
    <div class="book-reader-content">
      <component :is="viewerComponent" :filePath="bookPath" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent } from 'vue'
import { useRoute } from 'vue-router'
import TopNav from '@/components/home/TopNav.vue'
import type { NavLink } from '@/types/home'
import PdfViewer from '@/components/cirrus/viewers/PdfViewer.vue'

const navLinks: NavLink[] = [
  { name: 'Cirrus', href: '/cirrus' },
  { name: 'Photos', href: '/photos' },
  { name: 'Books', href: '/books' },
]

const route = useRoute()
const bookPath = computed(() => (route.query.path as string) || '')
const fileName = computed(() => bookPath.value.split('/').pop() || '')

function getFileType(path: string) {
  const ext = path.split('.').pop()?.toLowerCase()
  if (ext === 'pdf') return 'pdf'
  if (ext === 'epub') return 'epub'
  return 'unsupported'
}

const viewerComponent = computed(() => {
  const type = getFileType(bookPath.value)
  if (type === 'pdf') return PdfViewer
  return defineComponent({
    template: '<div class="error-text">Only PDF viewing is supported.</div>',
  })
})
</script>

<style lang="scss" scoped>
.book-reader-view {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--color-gray-50);
}
.book-reader-nav {
  display: flex;
  align-items: center;
  padding: var(--spacing-lg) var(--spacing-2xl);
  border-bottom: 1px solid var(--color-gray-200);
  background: white;
}
.book-reader-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: none;
  border: none;
  color: var(--color-primary-700);
  font-size: var(--font-size-base);
  cursor: pointer;
  font-weight: 600;
}
.book-reader-title {
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: var(--color-gray-900);
  margin-left: 1rem;
}
.book-reader-content {
  flex: 1;
  overflow: auto;
  background: var(--color-gray-100);
  display: flex;
  justify-content: center;
  align-items: flex-start;
}
</style>
