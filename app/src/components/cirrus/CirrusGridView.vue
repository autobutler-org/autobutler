<template>
  <div class="grid-view-container">
    <div class="grid-view-grid">
      <div
        v-for="file in files"
        :key="file.fullPath"
        :class="[
          'grid-view-item',
          'file-node',
          {
            'grid-view-item--folder': isDirectory(file),
            'grid-view-item--selected': selectedFile && selectedFile.fullPath === file.fullPath,
          },
        ]"
        :data-name="getFileName(file)"
        :data-is-folder="isDirectory(file)"
        :data-file-type="getFileType(file)"
        :data-device-name="file.deviceName"
        @click="emit('select', file)"
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
            <div
              v-if="props.showDeviceBadges && file.deviceName"
              class="device-badge"
              :title="'Device: ' + file.deviceName"
            >
              <svg
                class="device-badge-icon"
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
                <line x1="8" y1="21" x2="16" y2="21"></line>
                <line x1="12" y1="17" x2="12" y2="21"></line>
              </svg>
              <span class="device-badge-name">{{ file.deviceName }}</span>
            </div>
            <div v-if="!isDirectory(file)" class="grid-view-size">
              {{ formatBytes(getFileSize(file)) }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CirrusFileNode } from '@/types/cirrus'
import {
  determineFileType,
  getFileName,
  isDirectory,
  getFileSize,
  formatBytes,
} from '@/services/cirrusService'
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
  showDeviceBadges?: boolean
  selectedFile?: CirrusFileNode | null
}>()

const emit = defineEmits<{
  'navigate-folder': [path: string]
  'open-file': [file: CirrusFileNode]
  'context-menu': [event: MouseEvent, file: CirrusFileNode]
  select: [file: CirrusFileNode]
}>()

const getFileType = (file: CirrusFileNode) => determineFileType(file)

const getIconComponent = (fileType: string) => {
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

const handleClick = (file: CirrusFileNode) => {
  const fileName = getFileName(file)
  if (isDirectory(file)) {
    const newPath = props.currentPath ? `${props.currentPath}/${fileName}` : fileName
    emit('navigate-folder', newPath)
  } else {
    emit('open-file', file)
  }
}

const handleContextMenu = (event: MouseEvent, file: CirrusFileNode) => {
  event.preventDefault()
  emit('context-menu', event, file)
}
</script>

<style lang="scss" scoped>
.grid-view-container {
  flex: 1;
  overflow-y: auto;
  padding: $spacing-sm;
}

.grid-view-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: $spacing-md;
}

.grid-view-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $spacing-md;
  border-radius: $border-radius-lg;
  cursor: pointer;
  transition: background-color 0.15s ease;
  position: relative;

  &:hover {
    background-color: $color-gray-100;

    @media (prefers-color-scheme: dark) {
      background-color: $color-gray-800;
    }

    .context-menu-trigger {
      opacity: 1;
    }
  }
  &.grid-view-item--selected {
    background-color: $color-primary-100;
    @media (prefers-color-scheme: dark) {
      background-color: $color-primary-900;
    }
  }
}

.context-menu-trigger {
  position: absolute;
  top: $spacing-xs;
  right: $spacing-xs;
  background: transparent;
  border: none;
  cursor: pointer;
  padding: $spacing-xs;
  font-size: 1rem;
  color: $color-gray-600;
  border-radius: $border-radius-sm;
  opacity: 0;
  transition:
    opacity 0.15s ease,
    background-color 0.15s ease;

  &:hover {
    background-color: $color-gray-200;
  }

  @media (prefers-color-scheme: dark) {
    color: $color-gray-400;

    &:hover {
      background-color: $color-gray-700;
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
  margin-bottom: $spacing-sm;

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
  font-size: $font-size-sm;
  word-break: break-word;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
}

.grid-view-size {
  font-size: 0.75rem;
  color: $color-gray-500;
  margin-top: $spacing-xs;
}

/* Device badge - Shows which storage device a file is on */
.device-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  margin-top: $spacing-xs;
  padding: 2px 6px;
  background-color: $color-blue-50;
  border: 1px solid $color-blue-200;
  border-radius: $border-radius-sm;
  font-size: 10px;
  color: $color-blue-700;
  white-space: nowrap;

  @media (prefers-color-scheme: dark) {
    background-color: $color-blue-900;
    border-color: $color-blue-700;
    color: $color-blue-200;
  }
}

.device-badge-icon {
  width: 10px;
  height: 10px;
  flex-shrink: 0;
}

.device-badge-name {
  font-weight: 500;
  max-width: 60px;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
