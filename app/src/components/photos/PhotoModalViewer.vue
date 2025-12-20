<template>
  <div class="photo-viewer-modal-overlay" @click.self="close">
    <button class="photo-viewer-close" @click="close" aria-label="Close">&times;</button>
    <div class="photo-viewer-modal">
      <ImageViewer :filePath="path" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import ImageViewer from '@/components/cirrus/viewers/ImageViewer.vue'

const router = useRouter()
const route = useRoute()
const path = route.params.path as string

function close() {
  router.back()
}
</script>

<style scoped>
.photo-viewer-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.7);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
}
.photo-viewer-modal {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 16px rgba(0,0,0,0.18);
  padding: 0;
  max-width: 95vw;
  max-height: 95vh;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.photo-viewer-close {
  position: fixed;
  top: 2rem;
  right: 2rem;
  background: white;
  border: none;
  border-radius: 50%;
  font-size: 2.2rem;
  color: #333;
  cursor: pointer;
  z-index: 1100;
  box-shadow: 0 2px 8px rgba(0,0,0,0.10);
  width: 2.5rem;
  height: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
  padding: 0;
  line-height: 2.5rem;
}
.photo-viewer-close:hover {
  background: #f2f2f2;
}
</style>
