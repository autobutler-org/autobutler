<script setup lang="ts">
import type { CirrusFileNode } from '@/types/cirrus'
import { determineFileType, getFileName, isDirectory } from '@/services/cirrusService'
import FolderIcon from '@/components/icons/FolderIcon.vue'
import PdfIcon from '@/components/icons/PdfIcon.vue'
import ImageIcon from '@/components/icons/ImageIcon.vue'
import SlideshowIcon from '@/components/icons/SlideshowIcon.vue'
import ArchiveIcon from '@/components/icons/ArchiveIcon.vue'
import GenericIcon from '@/components/icons/GenericIcon.vue'
import DocxIcon from '@/components/icons/DocxIcon.vue'

const props = defineProps<{
  files: CirrusFileNode[]
  currentPath: string
}>()

const emit = defineEmits<{
  'navigate-folder': [path: string]
  'open-file': [file: CirrusFileNode]
}>()

function getFileType(file: CirrusFileNode) {
  return determineFileType(file)
}

function getIconComponent(fileType: string) {
  switch (fileType) {
    case 'folder':
      return FolderIcon
    case 'pdf':
      return PdfIcon
    case 'image':
      return ImageIcon
    case 'slideshow':
      return SlideshowIcon
    case 'archive':
      return ArchiveIcon
    case 'docx':
      return DocxIcon
    default:
      return GenericIcon
  }
}

function handleClick(file: CirrusFileNode) {
  const fileName = getFileName(file)
  if (isDirectory(file)) {
    const newPath = props.currentPath ? `${props.currentPath}/${fileName}` : fileName
    emit('navigate-folder', newPath)
  } else {
    emit('open-file', file)
  }
}
</script>

<template>
  <div class="grid-view-container">
    <div class="grid-view-grid">
      <div
        v-for="file in files"
        :key="file.fullPath"
        :class="['grid-view-item', 'file-node', { 'grid-view-item--folder': isDirectory(file) }]"
        :data-name="getFileName(file)"
        :data-is-folder="isDirectory(file)"
        :data-file-type="getFileType(file)"
        :data-device-name="file.deviceName"
        @dblclick="handleClick(file)"
      >
        <div class="grid-view-link">
          <div v-if="isDirectory(file)" class="grid-view-icon-container">
            <FolderIcon />
          </div>
          <div v-else class="grid-view-icon-container">
            <component :is="getIconComponent(getFileType(file))" />
          </div>
          <div class="grid-view-details">
            <div class="grid-view-name" :title="getFileName(file)">{{ getFileName(file) }}</div>
            <div v-if="!isDirectory(file)" class="grid-view-size">{{ getFileType(file) }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
