<template>
  <div class="file-table-container">
    <table id="file-explorer-table" class="file-table">
      <thead class="file-table-header">
        <tr>
          <CirrusListViewSortHeader
            v-for="column in sortColumns"
            :key="column.column ? column.column : ''"
            :header="column.column"
            :active-sort-column="sortColumn"
            :sort-direction="sortDirection"
            :align-direction="column.alignDirection"
            @toggle:sort="toggleSort"
          />
          <th class="file-table-header-cell file-table-header-cell--toggle">
            <button
              class="sort-switcher"
              type="button"
              :title="mixedSorting ? 'Switch to Folders First sorting' : 'Switch to Mixed sorting'"
              @click="toggleMixedSorting"
            >
              <SortSwitcherIcon :mixed-sorting="mixedSorting" />
            </button>
          </th>
        </tr>
      </thead>
      <tbody id="file-explorer-list" class="file-table-body">
        <tr
          v-for="file in sortedFiles"
          :key="file.fullPath"
          class="file-table-row file-node"
          :class="{
            'file-table-row--selected': selectedFile && selectedFile.fullPath === file.fullPath,
          }"
          :data-name="getFileName(file)"
          :data-file-type="determineFileType(file)"
          :data-device-name="file.deviceName"
          @click="emit('select', file)"
          @dblclick="handleClick(file)"
          @contextmenu="handleContextMenu($event, file)"
        >
          <template v-if="isDirectory(file)">
            <td class="file-table-cell file-table-cell--content">
              <FolderIcon />
              <span class="file-table-name">{{ getFileName(file) }}</span>
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
          <template v-else>
            <td class="file-table-cell file-table-cell--clickable">
              <component :is="getIconComponent(determineFileType(file))" />
              <span class="file-table-name">{{ getFileName(file) }}</span>
              <span
                v-if="props.showDeviceBadges && file.deviceName"
                class="device-badge"
                :title="'Device: ' + file.deviceName"
              >
                <DeviceBadge :device-name="file.deviceName" />
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
import CirrusListViewSortHeader, {
  type HeaderAlignDirection,
  type SortColumn,
  type SortDirection,
} from './CirrusListViewSortHeader.vue'
import DeviceBadge from '@/components/badges/DeviceBadge.vue'
import SortSwitcherIcon from '@/components/icons/SortSwitcherIcon.vue'

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

// Sorting state
const sortColumn = ref<SortColumn>(null)
const sortDirection = ref<SortDirection>('asc')
const mixedSorting = ref(false)
const sortColumns: { column: SortColumn; alignDirection?: HeaderAlignDirection }[] = [
  {
    column: 'name',
    alignDirection: 'left',
  },
  { column: 'size' },
]

// Toggle mixed sorting mode (folders mixed with files vs folders first)
const toggleMixedSorting = () => {
  mixedSorting.value = !mixedSorting.value
}

// Sorted files computed property
// TODO: Move the sorting into a super generic utility module, which allows you to sort by "sections" (e.g., folders first)
// or as a whole, accepting predicates for the "sections"
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

const toggleSort = (column: SortColumn): void => {
  if (sortColumn.value === column) {
    // Toggle direction
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    // New column, start ascending
    sortColumn.value = column
    sortDirection.value = 'asc'
  }
}

// TODO: Move this to a utility module
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
  border-radius: $border-radius-lg;
}

.file-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.file-table-header {
  background-color: $color-gray-50;
  position: sticky;
  top: 0;

  @media (prefers-color-scheme: dark) {
    background-color: $color-gray-900;
  }
}

.file-table-header-cell {
  height: 3rem;
  padding: 0 $spacing-sm;
  font-weight: 600;
  color: $color-gray-700;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-300;
  }

  &--toggle {
    width: 4rem;
  }
}

.file-table-body {
  border-top: 1px solid $color-gray-200;

  tr {
    border-top: 1px solid $color-gray-200;
  }

  @media (prefers-color-scheme: dark) {
    border-color: $color-gray-700;

    tr {
      border-color: $color-gray-700;
    }
  }
}

.file-table-row {
  cursor: pointer;

  &:hover {
    background-color: $color-gray-100;

    .file-table-cell .file-table-name {
      text-decoration: underline;
    }

    @media (prefers-color-scheme: dark) {
      background-color: $color-gray-800;
    }
  }
  &.file-table-row--selected {
    background-color: $color-primary-100;

    @media (prefers-color-scheme: dark) {
      background-color: $color-primary-900;
    }

    .file-table-cell .file-table-name {
      text-decoration: underline;
    }
  }
}

.file-table-cell {
  padding: 0;

  &--content {
    display: flex;
    align-items: center;
    padding: $spacing-sm 0 $spacing-sm $spacing-sm;
    height: 100%;
  }

  &--clickable {
    display: flex;
    align-items: center;
    padding: $spacing-sm 0 $spacing-sm $spacing-sm;
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
  border-radius: $border-radius;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 1.5rem;
  color: $color-gray-500;

  &:hover {
    background-color: $color-gray-200;
    color: $color-gray-700;
  }

  @media (prefers-color-scheme: dark) {
    color: $color-gray-400;

    &:hover {
      background-color: $color-gray-700;
      color: $color-gray-200;
    }
  }
}

.file-table-name {
  flex: 1;
  margin-left: $spacing-sm;
  color: inherit;
  display: flex;
  align-items: center;
}

.file-table-size {
  color: $color-gray-500;
  font-size: $font-size-sm;
  text-align: right;
  padding-right: $spacing-sm;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-200;
  }
}

.sort-switcher {
  width: 100%;
  height: 3rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: $spacing-xs;
  padding-left: $spacing-sm;
  border: none;
  background: transparent;
  font-size: $font-size-xs;
  cursor: pointer;
  border-radius: $border-radius;

  &:hover {
    background-color: $color-gray-200;
  }

  @media (prefers-color-scheme: dark) {
    &:hover {
      background-color: $color-gray-700;
    }
  }
}

/* Device badge - Shows which storage device a file is on */
.device-badge {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
  margin-left: $spacing-sm;
  padding: 2px 6px;
  background-color: $color-blue-50;
  border: 1px solid $color-blue-200;
  border-radius: $border-radius-sm;
  font-size: $font-size-xs;
  color: $color-blue-700;
  white-space: nowrap;

  @media (prefers-color-scheme: dark) {
    background-color: $color-blue-900;
    border-color: $color-blue-700;
    color: $color-blue-200;
  }
}
</style>
