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
  'context-menu': [event: MouseEvent, file: CirrusFileNode]
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

function handleContextMenu(event: MouseEvent, file: CirrusFileNode) {
  event.preventDefault()
  emit('context-menu', event, file)
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
        @contextmenu="handleContextMenu($event, file)"
      >
        <button
          class="context-menu-trigger"
          type="button"
          title="More actions"
          @click.stop="handleContextMenu($event, file)"
        >
          &#x22EE;
        </button>
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

<style lang="scss" scoped>
.grid-view-container {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-sm);
}

.grid-view-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: var(--spacing-md);
}

.grid-view-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--spacing-md);
  border-radius: var(--border-radius-lg);
  cursor: pointer;
  transition: background-color 0.15s ease;
  position: relative;

  &:hover {
    background-color: var(--color-gray-100);

    @media (prefers-color-scheme: dark) {
      background-color: var(--color-gray-800);
    }

    .context-menu-trigger {
      opacity: 1;
    }
  }
}

.context-menu-trigger {
  position: absolute;
  top: var(--spacing-xs);
  right: var(--spacing-xs);
  background: transparent;
  border: none;
  cursor: pointer;
  padding: var(--spacing-xs);
  font-size: 1rem;
  color: var(--color-gray-600);
  border-radius: var(--border-radius-sm);
  opacity: 0;
  transition: opacity 0.15s ease, background-color 0.15s ease;

  &:hover {
    background-color: var(--color-gray-200);
  }

  @media (prefers-color-scheme: dark) {
    color: var(--color-gray-400);

    &:hover {
      background-color: var(--color-gray-700);
    }
  }
}

.grid-view-link {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-decoration: none;
  color: inherit;
  width: 100%;
}

.grid-view-icon-container {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: var(--spacing-sm);

  svg {
    width: 100%;
    height: 100%;
  }
}

.grid-view-details {
  text-align: center;
  width: 100%;
}

.grid-view-name {
  font-size: var(--font-size-sm);
  word-break: break-word;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
}

.grid-view-size {
  font-size: 0.75rem;
  color: var(--color-gray-500);
  margin-top: var(--spacing-xs);
}
</style>
