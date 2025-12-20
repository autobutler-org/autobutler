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
  'context-menu': [event: MouseEvent, file: CirrusFileNode]
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

function handleContextMenu(event: MouseEvent, file: CirrusFileNode) {
  event.preventDefault()
  emit('context-menu', event, file)
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
          @contextmenu="handleContextMenu($event, file)"
        >
          <template v-if="isDirectory(file)">
            <td class="file-table-cell file-table-cell--content">
              <FolderIcon />
              <span class="file-table-name">{{ getFileName(file) }}</span>
            </td>
            <td class="file-table-cell file-table-size">
              Folder
            </td>
            <td class="file-table-cell file-table-cell--menu">
              <button
                type="button"
                class="context-menu-trigger"
                aria-label="Open context menu"
                @click.stop="handleContextMenu($event, file)"
              >
                &#x22EE;
              </button>
            </td>
          </template>
          <template v-else>
            <td class="file-table-cell file-table-cell--clickable">
              <component :is="getIconComponent(getFileType(file))" />
              <span class="file-table-name">{{ getFileName(file) }}</span>
            </td>
            <td class="file-table-cell file-table-size">
              {{ getFileType(file) }}
            </td>
            <td class="file-table-cell file-table-cell--menu">
              <button
                type="button"
                class="context-menu-trigger"
                aria-label="Open context menu"
                @click.stop="handleContextMenu($event, file)"
              >
                &#x22EE;
              </button>
            </td>
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

<style lang="scss" scoped>
.file-table-container {
  flex: 1;
  min-height: 0;
  position: relative;
  overflow-y: auto;
  overflow-x: hidden;
  border-radius: var(--border-radius-lg);
}

.file-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.file-table-header {
  background-color: var(--color-gray-50);
  position: sticky;
  top: 0;

  @media (prefers-color-scheme: dark) {
    background-color: var(--color-gray-900);
  }
}

.file-table-header-cell {
  height: 3rem;
  padding: 0 var(--spacing-sm);
  font-weight: 600;
  color: var(--color-gray-700);

  @media (prefers-color-scheme: dark) {
    color: var(--color-gray-300);
  }

  &--left {
    text-align: left;
  }

  &--right {
    text-align: right;
    width: 6rem;
  }

  &--toggle {
    width: 4rem;
  }
}

.file-table-body {
  border-top: 1px solid var(--color-gray-200);

  tr {
    border-top: 1px solid var(--color-gray-200);
  }

  @media (prefers-color-scheme: dark) {
    border-color: var(--color-gray-700);

    tr {
      border-color: var(--color-gray-700);
    }
  }
}

.file-table-row {
  cursor: pointer;

  &:hover {
    background-color: var(--color-gray-100);

    @media (prefers-color-scheme: dark) {
      background-color: var(--color-gray-800);
    }
  }
}

.file-table-cell {
  padding: 0;

  &--content {
    display: flex;
    align-items: center;
    padding: var(--spacing-sm) 0 var(--spacing-sm) var(--spacing-sm);
    height: 100%;
  }

  &--clickable {
    display: flex;
    align-items: center;
    padding: var(--spacing-sm) 0 var(--spacing-sm) var(--spacing-sm);
    height: 100%;
    cursor: pointer;
  }

  &--menu {
    width: 60px;
    text-align: center;
    vertical-align: middle;
  }

  &--spacer {
    text-align: center;
    font-style: italic;
    color: var(--color-gray-400);
    cursor: pointer;
    padding: var(--spacing-sm) 0;

    &:hover {
      background-color: var(--color-gray-100);

      @media (prefers-color-scheme: dark) {
        background-color: var(--color-gray-800);
      }
    }
  }
}

.context-menu-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  padding: 0;
  border-radius: var(--border-radius);
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 1.5rem;
  color: var(--color-gray-500);

  &:hover {
    background-color: var(--color-gray-200);
    color: var(--color-gray-700);
  }

  @media (prefers-color-scheme: dark) {
    color: var(--color-gray-400);

    &:hover {
      background-color: var(--color-gray-700);
      color: var(--color-gray-200);
    }
  }
}

.file-table-name {
  flex: 1;
  margin-left: var(--spacing-sm);
  color: inherit;
  display: flex;
  align-items: center;
}

.file-table-size {
  color: var(--color-gray-500);
  font-size: var(--font-size-sm);
  text-align: right;
  padding-right: var(--spacing-sm);

  @media (prefers-color-scheme: dark) {
    color: var(--color-gray-200);
  }
}
</style>
