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
        :data-file-type="determineFileType(file)"
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
            <CirrusFolderIcon />
          </div>
          <div v-else class="grid-view-icon-container">
            <component :is="getIconComponent(determineFileType(file))" />
          </div>
          <div class="grid-view-details">
            <div class="grid-view-name" :title="getFileName(file)">{{ getFileName(file) }}</div>
            <DeviceBadge
              v-if="props.showDeviceBadges && file.deviceName"
              :device-name="file.deviceName"
            />
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
import CirrusFolderIcon from '@/components/icons/CirrusFolderIcon.vue'
import PdfIcon from '@/components/icons/PdfIcon.vue'
import ImageIcon from '@/components/icons/ImageIcon.vue'
import SlideshowIcon from '@/components/icons/SlideshowIcon.vue'
import ArchiveIcon from '@/components/icons/ArchiveIcon.vue'
import GenericIcon from '@/components/icons/GenericIcon.vue'
import DocxIcon from '@/components/icons/DocxIcon.vue'
import DeviceBadge from '@/components/badges/DeviceBadge.vue'

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

// TODO: CirrusListView has the exact same functions/code, after this point

// TODO: Move this to a utility module
const getIconComponent = (fileType: string) => {
  switch (fileType) {
    case 'folder':
      return CirrusFolderIcon
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
</style>
