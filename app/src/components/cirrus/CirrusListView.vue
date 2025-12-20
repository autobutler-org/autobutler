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
import { type Component } from 'vue'

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

function getIconComponent(fileType: string): Component {
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
    // Navigate to folder
    const newPath = props.currentPath ? `${props.currentPath}/${fileName}` : fileName
    emit('navigate-folder', newPath)
  } else {
    emit('open-file', file)
  }
}
</script>

<template>
  <div class="file-table-container">
    <table id="file-explorer-table" class="file-table">
      <thead class="file-table-header">
        <tr>
          <th class="file-table-header-cell file-table-header-cell--left">Name</th>
          <th class="file-table-header-cell file-table-header-cell--right">Type</th>
          <th class="file-table-header-cell file-table-header-cell--toggle"></th>
        </tr>
      </thead>
      <tbody id="file-explorer-list" class="file-table-body">
        <tr
          v-for="file in files"
          :key="file.fullPath"
          class="file-table-row file-node"
          :data-name="getFileName(file)"
          :data-file-type="getFileType(file)"
          :data-device-name="file.deviceName"
          @dblclick="handleClick(file)"
        >
          <template v-if="isDirectory(file)">
            <td class="file-table-cell file-table-cell--content">
              <FolderIcon />
              <span class="file-table-name">{{ getFileName(file) }}</span>
            </td>
            <td class="file-table-cell file-table-size">
              Folder
            </td>
            <td class="file-table-cell"></td>
          </template>
          <template v-else>
            <td class="file-table-cell file-table-cell--clickable">
              <component :is="getIconComponent(getFileType(file))" />
              <span class="file-table-name">{{ getFileName(file) }}</span>
            </td>
            <td class="file-table-cell file-table-size">
              {{ getFileType(file) }}
            </td>
            <td class="file-table-cell"></td>
          </template>
        </tr>
        <!-- Spacer row for drag and drop hint -->
        <tr class="file-node">
          <td colspan="3" class="file-table-cell file-table-cell--spacer">
            <span class="spacer">Drop files here…</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
