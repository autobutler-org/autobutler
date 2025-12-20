<script setup lang="ts">
import type { DeviceFileInfo } from '@/types/cirrus'
import { determineFileType, formatBytes } from '@/services/cirrusService'
import FolderIcon from '@/components/icons/FolderIcon.vue'
import PdfIcon from '@/components/icons/PdfIcon.vue'
import ImageIcon from '@/components/icons/ImageIcon.vue'
import SlideshowIcon from '@/components/icons/SlideshowIcon.vue'
import ArchiveIcon from '@/components/icons/ArchiveIcon.vue'
import GenericIcon from '@/components/icons/GenericIcon.vue'
import DocxIcon from '@/components/icons/DocxIcon.vue'

const props = defineProps<{
  files: DeviceFileInfo[]
  currentPath: string
}>()

const emit = defineEmits<{
  'navigate-folder': [path: string]
  'open-file': [file: DeviceFileInfo]
}>()

function getFileType(file: DeviceFileInfo) {
  return determineFileType(file.name, file.isDir)
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

function handleClick(file: DeviceFileInfo) {
  if (file.isDir) {
    const newPath = props.currentPath ? `${props.currentPath}/${file.name}` : file.name
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
        :key="file.name"
        :class="['grid-view-item', 'file-node', { 'grid-view-item--folder': file.isDir }]"
        :data-name="file.name"
        :data-is-folder="file.isDir"
        :data-file-type="getFileType(file)"
        :data-device-name="file.deviceName"
        @dblclick="handleClick(file)"
      >
        <div class="grid-view-link">
          <div v-if="file.isDir" class="grid-view-icon-container">
            <FolderIcon />
          </div>
          <div v-else class="grid-view-icon-container">
            <component :is="getIconComponent(getFileType(file))" />
          </div>
          <div class="grid-view-details">
            <div class="grid-view-name" :title="file.name">{{ file.name }}</div>
            <div v-if="!file.isDir" class="grid-view-size">{{ formatBytes(file.size) }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
