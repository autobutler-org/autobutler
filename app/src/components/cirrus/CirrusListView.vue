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
              :title="
                mixedSorting
                  ? 'Switch to Folders First sorting'
                  : 'Switch to Mixed sorting'
              "
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
            'file-table-row--selected':
              selectedFile && selectedFile.fullPath === file.fullPath,
          }"
          :data-name="getFileName(file)"
          :data-file-type="determineFileType(file)"
          :data-device-name="file.deviceName"
          @click="emit('select', file)"
          @dblclick="handleClick(file)"
          @contextmenu="handleContextMenu($event, file)"
        >
          <td class="file-table-cell file-table-cell--clickable">
            <component :is="getIconComponent(determineFileType(file))" />
            <span class="file-table-name">{{ getFileName(file) }}</span>
            <DeviceBadge
              v-if="props.showDeviceBadges && file.deviceName"
              :device-name="file.deviceName"
            />
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
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script lang="ts" setup>
import DeviceBadge from '@/components/badges/DeviceBadge.vue'
import ArchiveIcon from '@/components/icons/ArchiveIcon.vue'
import CirrusFolderIcon from '@/components/icons/CirrusFolderIcon.vue'
import DocxIcon from '@/components/icons/DocxIcon.vue'
import GenericIcon from '@/components/icons/GenericIcon.vue'
import ImageIcon from '@/components/icons/ImageIcon.vue'
import PdfIcon from '@/components/icons/PdfIcon.vue'
import SlideshowIcon from '@/components/icons/SlideshowIcon.vue'
import SortSwitcherIcon from '@/components/icons/SortSwitcherIcon.vue'
import {
  determineFileType,
  formatBytes,
  getFileName,
  getFileSize,
  isDirectory,
} from '@/services/cirrusService'
import type { CirrusFileNode } from '@/types/cirrus'
import { computed, ref, type Component } from 'vue'
import CirrusListViewSortHeader, {
  type HeaderAlignDirection,
  type SortColumn,
  type SortDirection,
} from './CirrusListViewSortHeader.vue'

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
const sortColumns: {
  column: SortColumn
  alignDirection?: HeaderAlignDirection
}[] = [
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

// TODO: CirrusGridView has the exact same functions/code, after this point

// TODO: Move this to a utility module
const getIconComponent = (fileType: string): Component => {
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
    // Navigate to folder
    const newPath = props.currentPath
      ? `${props.currentPath}/${fileName}`
      : fileName
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
  background-color: $theme-palette-bg-nav;
  position: sticky;
  top: 0;
}

.file-table-header-cell {
  height: 3rem;
  padding: 0 $spacing-sm;
  font-weight: 600;
  color: $theme-palette-text-primary;

  &:hover {
    background-color: $color-gray-100;

    @media (prefers-color-scheme: dark) {
      background-color: $color-gray-800;
    }
  }

  &--toggle {
    width: 4rem;
  }
}

.file-table-body {
  border-top: 1px solid $theme-palette-border;

  tr {
    border-top: 1px solid $theme-palette-border;
  }
}

.file-table-row {
  cursor: pointer;

  &:hover {
    background-color: $theme-palette-bg-secondary;

    .file-table-cell .file-table-name {
      text-decoration: underline;
    }
  }
  &.file-table-row--selected {
    background-color: $theme-palette-accent;

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
  color: $theme-palette-text-muted;

  &:hover {
    background-color: $theme-palette-accent-hover;
    color: $theme-palette-text-inverse;
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
  color: $theme-palette-text-muted;
  font-size: $theme-font-size-sm;
  text-align: right;
  padding-right: $spacing-sm;
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
  font-size: $theme-font-size-xs;
  cursor: pointer;
  border-radius: $border-radius;
}
</style>
