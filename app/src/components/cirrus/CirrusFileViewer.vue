<template>
  <ModalDialog v-if="modelValue" @close="closeDialog">
    <div class="file-viewer-content">
      <component :is="viewerComponent" :file-path="filePath" />
    </div>
  </ModalDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FileType } from '@/types/cirrus'
import ImageViewer from './viewers/ImageViewer.vue'
import VideoViewer from './viewers/VideoViewer.vue'
import PdfViewer from './viewers/PdfViewer.vue'
import TextViewer from './viewers/TextViewer.vue'
import UnsupportedViewer from './viewers/UnsupportedViewer.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'

const props = defineProps<{
  modelValue: boolean
  filePath: string
  fileType: FileType
}>()
const emit = defineEmits(['update:modelValue'])

// Determine which viewer component to use
const viewerComponent = computed(() => {
  switch (props.fileType) {
    case 'image':
      return ImageViewer
    case 'video':
      return VideoViewer
    case 'pdf':
      return PdfViewer
    case 'generic':
      return TextViewer
    case 'epub':
    case 'docx':
    case 'archive':
    case 'slideshow':
    default:
      return UnsupportedViewer
  }
})

const closeDialog = () => {
  emit('update:modelValue', false)
}
</script>

<style lang="scss" scoped>
.file-viewer-content {
  margin: var(--spacing-3xl);
  width: calc(100% - (var(--spacing-3xl) * 2));
  height: calc(100% - (var(--spacing-3xl) * 2));
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
</style>
