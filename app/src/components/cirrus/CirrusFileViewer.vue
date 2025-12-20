<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import type { FileType } from '@/types/cirrus'
import ImageViewer from './viewers/ImageViewer.vue'
import VideoViewer from './viewers/VideoViewer.vue'
import PdfViewer from './viewers/PdfViewer.vue'
import TextViewer from './viewers/TextViewer.vue'
import UnsupportedViewer from './viewers/UnsupportedViewer.vue'

const props = defineProps<{
  modelValue: boolean
  filePath: string
  fileType: FileType
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const dialogRef = ref<HTMLDialogElement | null>(null)

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})

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

function closeDialog() {
  isOpen.value = false
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    closeDialog()
  }
}

function handleBackdropClick(event: MouseEvent) {
  // Close if clicking on the dialog backdrop (not the content)
  if (event.target === dialogRef.value) {
    closeDialog()
  }
}

// Watch for open/close state changes
watch(isOpen, (newValue) => {
  if (newValue) {
    dialogRef.value?.showModal()
  } else {
    dialogRef.value?.close()
  }
})

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <dialog
    ref="dialogRef"
    class="file-viewer-dialog"
    @click="handleBackdropClick"
    @close="closeDialog"
  >
    <button class="file-viewer-close" aria-label="Close" @click="closeDialog">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        fill="none"
        viewBox="0 0 20 20"
      >
        <path
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          d="M5 5l10 10M15 5l-10 10"
        />
      </svg>
    </button>
    <div class="file-viewer-content" @click.stop>
      <component :is="viewerComponent" :file-path="filePath" />
    </div>
  </dialog>
</template>

<style lang="scss" scoped>
.file-viewer-dialog {
  width: 100vw;
  height: 100vh;
  max-width: 100vw;
  max-height: 100vh;
  padding: 0;
  margin: 0;
  border: none;
  background: transparent;
  inset: 0;
  background-color: var(--color-gray-100);
  border-radius: var(--border-radius-lg);
  box-shadow: var(--shadow-xl);
  overflow: hidden;

  &::backdrop {
    background: rgba(0, 0, 0, 0.8);
  }

  @media (prefers-color-scheme: dark) {
    background-color: var(--color-gray-900);
  }
}

.file-viewer-close {
  position: absolute;
  top: var(--spacing-lg);
  right: var(--spacing-lg);
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  padding: 0;
  background: rgba(0, 0, 0, 0.5);
  border: none;
  border-radius: 50%;
  color: white;
  cursor: pointer;
  transition: background-color 0.2s ease;

  &:hover {
    background: rgba(0, 0, 0, 0.7);
  }
}

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
