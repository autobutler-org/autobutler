<template>
  <div class="file-table-container">
    <table id="file-explorer-table" class="file-table">
      <thead class="file-table-header">
        <tr>
          <th
            class="file-table-header-cell file-table-header-cell--left file-table-header-cell--sortable"
            @click="toggleSort('name')"
          >
            <span class="sort-button">
              <span>Name</span>
              <span class="sort-arrows">
                <svg
                  v-if="sortColumn === 'name' && sortDirection === 'asc'"
                  class="sort-arrow sort-arrow--active"
                  fill="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path d="M7 14l5-5 5 5z"></path>
                </svg>
                <svg
                  v-else-if="sortColumn === 'name' && sortDirection === 'desc'"
                  class="sort-arrow sort-arrow--active"
                  fill="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path d="M7 10l5 5 5-5z"></path>
                </svg>
                <svg v-else class="sort-arrow" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M7 8l5-5 5 5z"></path>
                  <path d="M7 16l5 5 5-5z"></path>
                </svg>
              </span>
            </span>
          </th>
          <th
            class="file-table-header-cell file-table-header-cell--right file-table-header-cell--sortable"
            @click="toggleSort('size')"
          >
            <span class="sort-button">
              <span>Size</span>
              <span class="sort-arrows">
                <svg
                  v-if="sortColumn === 'size' && sortDirection === 'asc'"
                  class="sort-arrow sort-arrow--active"
                  fill="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path d="M7 14l5-5 5 5z"></path>
                </svg>
                <svg
                  v-else-if="sortColumn === 'size' && sortDirection === 'desc'"
                  class="sort-arrow sort-arrow--active"
                  fill="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path d="M7 10l5 5 5-5z"></path>
                </svg>
                <svg v-else class="sort-arrow" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M7 8l5-5 5 5z"></path>
                  <path d="M7 16l5 5 5-5z"></path>
                </svg>
              </span>
            </span>
          </th>
          <th class="file-table-header-cell file-table-header-cell--toggle">
            <button
              class="sort-switcher"
              type="button"
              :title="mixedSorting ? 'Switch to Folders First sorting' : 'Switch to Mixed sorting'"
              @click="toggleMixedSorting"
            >
              <div class="sort-switcher-icons">
                <svg class="sort-switcher-arrows" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M7 8l5-5 5 5z"></path>
                  <path d="M7 16l5 5 5-5z"></path>
                </svg>
                <svg
                  v-if="!mixedSorting"
                  class="sort-switcher-folder"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z"
                  ></path>
                </svg>
                <svg v-else class="sort-switcher-file" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fill-rule="evenodd"
                    d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z"
                    clip-rule="evenodd"
                  ></path>
                </svg>
              </div>
              <span class="sort-switcher-label">{{ mixedSorting ? 'Mixed' : 'Folders' }}</span>
            </button>
          </th>
        </tr>
      </thead>
      <tbody id="file-explorer-list" class="file-table-body">
        <tr
          v-for="file in sortedFiles"
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
              <span
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
              </span>
            </td>
            <td class="file-table-cell file-table-size">—</td>
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
              <span
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
              </span>
            </td>
            <td class="file-table-cell file-table-size">
              {{ formatBytes(getFileSize(file)) }}
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
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
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
import { type Component } from 'vue'

const props = defineProps<{
  files: CirrusFileNode[]
  currentPath: string
  showDeviceBadges?: boolean
}>()

const emit = defineEmits<{
  'navigate-folder': [path: string]
  'open-file': [file: CirrusFileNode]
  'context-menu': [event: MouseEvent, file: CirrusFileNode]
}>()

// Sorting state
type SortColumn = 'name' | 'size' | null
type SortDirection = 'asc' | 'desc'

const sortColumn = ref<SortColumn>(null)
const sortDirection = ref<SortDirection>('asc')
const mixedSorting = ref(false)

// Toggle mixed sorting mode (folders mixed with files vs folders first)
const toggleMixedSorting = () => {
  mixedSorting.value = !mixedSorting.value
}

// Sorted files computed property
const sortedFiles = computed(() => {
  if (!sortColumn.value) {
    // Default: folders first (unless mixed), then alphabetically by name
    return [...props.files].sort((a, b) => {
      const aIsDir = isDirectory(a)
      const bIsDir = isDirectory(b)

      // Folders first unless mixed sorting is enabled
      if (!mixedSorting.value) {
        if (aIsDir && !bIsDir) return -1
        if (!aIsDir && bIsDir) return 1
      }

      return getFileName(a).localeCompare(getFileName(b), undefined, {
        numeric: true,
        sensitivity: 'base',
      })
    })
  }

  return [...props.files].sort((a, b) => {
    const aIsDir = isDirectory(a)
    const bIsDir = isDirectory(b)

    // Folders first unless mixed sorting is enabled
    if (!mixedSorting.value) {
      if (aIsDir && !bIsDir) return -1
      if (!aIsDir && bIsDir) return 1
    }

    let comparison = 0

    if (sortColumn.value === 'name') {
      comparison = getFileName(a).localeCompare(getFileName(b), undefined, {
        numeric: true,
        sensitivity: 'base',
      })
    } else if (sortColumn.value === 'size') {
      comparison = getFileSize(a) - getFileSize(b)
    }

    return sortDirection.value === 'asc' ? comparison : -comparison
  })
})

const toggleSort = (column: SortColumn) => {
  if (sortColumn.value === column) {
    // Toggle direction
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    // New column, start ascending
    sortColumn.value = column
    sortDirection.value = 'asc'
  }
}

const getFileType = (file: CirrusFileNode) => determineFileType(file)

const getIconComponent = (fileType: string): Component => {
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
    // Navigate to folder
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

  &--sortable {
    cursor: pointer;
    user-select: none;

    &:hover {
      background-color: var(--color-gray-100);

      @media (prefers-color-scheme: dark) {
        background-color: var(--color-gray-800);
      }
    }
  }
}

.sort-button {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-xs);
}

.sort-arrows {
  display: inline-flex;
  align-items: center;
}

.sort-arrow {
  width: 16px;
  height: 16px;
  color: var(--color-gray-400);

  &--active {
    color: var(--color-gray-700);

    @media (prefers-color-scheme: dark) {
      color: var(--color-gray-300);
    }
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

.sort-switcher {
  width: 100%;
  height: 3rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-xs);
  padding-left: var(--spacing-sm);
  border: none;
  background: transparent;
  font-size: var(--font-size-xs);
  cursor: pointer;
  border-radius: var(--border-radius);

  &:hover {
    background-color: var(--color-gray-200);
  }

  @media (prefers-color-scheme: dark) {
    &:hover {
      background-color: var(--color-gray-700);
    }
  }
}

.sort-switcher-icons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-xs);
  margin-bottom: var(--spacing-xs);
  height: 1rem;
  padding-left: var(--spacing-xs);
}

.sort-switcher-arrows,
.sort-switcher-folder,
.sort-switcher-file {
  width: 1rem;
  height: 1rem;
  color: var(--color-gray-500);

  @media (prefers-color-scheme: dark) {
    color: var(--color-gray-400);
  }
}

.sort-switcher-label {
  color: var(--color-gray-500);
  font-weight: 600;
  width: 3rem;
  text-align: center;

  @media (prefers-color-scheme: dark) {
    color: var(--color-gray-400);
  }
}

/* Device badge - Shows which storage device a file is on */
.device-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-xs);
  margin-left: var(--spacing-sm);
  padding: 2px 6px;
  background-color: var(--color-blue-50);
  border: 1px solid var(--color-blue-200);
  border-radius: var(--border-radius-sm);
  font-size: var(--font-size-xs);
  color: var(--color-blue-700);
  white-space: nowrap;

  @media (prefers-color-scheme: dark) {
    background-color: var(--color-blue-900);
    border-color: var(--color-blue-700);
    color: var(--color-blue-200);
  }
}

.device-badge-icon {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
}

.device-badge-name {
  font-weight: 500;
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
