<template>
  <div
    class="file-table-container"
    :class="{ 'file-table-container--dragging': isDragOver }"
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <div class="file-table-drop-overlay" v-show="isDragOver" />
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
          :data-name="CirrusService.getFileName(file)"
          :data-file-type="CirrusService.determineFileType(file)"
          :data-device-name="file.deviceName"
          @click="emit('select', file)"
          @dblclick="handleClick(file)"
          @contextmenu="handleContextMenu($event, file)"
          @dragenter="handleDirectoryDragEnter($event, file)"
          @dragover="handleDirectoryDragOver($event, file)"
          @dragleave="handleDirectoryDragLeave($event, file)"
          @drop="handleDirectoryDrop($event, file)"
        >
          <td class="file-table-cell file-table-cell--clickable">
            <div
              class="file-table-name-container"
              :class="{
                'file-table-name-container--drop-target':
                  CirrusService.isDirectory(file) &&
                  hoveredDirectoryPath === resolveDirectoryTargetPath(file),
              }"
            >
              <component
                :is="getIconComponent(CirrusService.determineFileType(file))"
              />
              <span class="file-table-name">
                <span class="file-table-name-label">
                  {{ CirrusService.getFileName(file) }}
                </span>
              </span>
            </div>
            <DeviceBadge
              v-if="props.showDeviceBadges && file.deviceName"
              :device-name="file.deviceName"
            />
          </td>
          <td class="file-table-cell file-table-size">
            {{ CirrusService.formatBytes(CirrusService.getFileSize(file)) }}
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
import { useCirrusFileDropZone } from '@/composables/useCirrusFileDropZone'
import CirrusService from '@/services/cirrusService'
import type { CirrusFileNode } from '@/types/cirrus'
import { computed, ref, watch, type Component } from 'vue'
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
  'files-uploaded': [files: CirrusFileNode[]]
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

const {
  isDragOver,
  handleDragEnter,
  handleDragOver,
  handleDragLeave,
  handleDrop,
} = useCirrusFileDropZone({
  currentPath: computed(() => props.currentPath),
  onFilesUploaded: (files) => emit('files-uploaded', files),
})

const hoveredDirectoryPath = ref<string | null>(null)

const normalizeCurrentPath = computed(() =>
  CirrusService.normalizePath(props.currentPath),
)

const resolveDirectoryTargetPath = (file: CirrusFileNode) => {
  const directoryName = CirrusService.getFileName(file)
  const basePath = normalizeCurrentPath.value
  return basePath ? `${basePath}/${directoryName}` : directoryName
}

const clearHoveredDirectory = () => {
  hoveredDirectoryPath.value = null
}

watch(isDragOver, (active) => {
  if (!active) {
    clearHoveredDirectory()
  }
})

const handleDirectoryDragEnter = (event: DragEvent, file: CirrusFileNode) => {
  if (!CirrusService.isDirectory(file)) return
  event.preventDefault()
  hoveredDirectoryPath.value = resolveDirectoryTargetPath(file)
}

const handleDirectoryDragOver = (event: DragEvent, file: CirrusFileNode) => {
  if (!CirrusService.isDirectory(file)) return
  event.preventDefault()
  hoveredDirectoryPath.value = resolveDirectoryTargetPath(file)
}

const handleDirectoryDragLeave = (event: DragEvent, file: CirrusFileNode) => {
  if (!CirrusService.isDirectory(file)) return
  event.preventDefault()

  const currentTarget = event.currentTarget as Node | null
  const relatedTarget = event.relatedTarget as Node | null

  if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget)) {
    return
  }

  if (hoveredDirectoryPath.value === resolveDirectoryTargetPath(file)) {
    clearHoveredDirectory()
  }
}

const handleDirectoryDrop = async (event: DragEvent, file: CirrusFileNode) => {
  if (!CirrusService.isDirectory(file)) return
  event.preventDefault()
  event.stopPropagation()
  const targetPath = resolveDirectoryTargetPath(file)
  clearHoveredDirectory()
  await handleDrop(event, targetPath)
}

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
      const aIsDir = CirrusService.isDirectory(a)
      const bIsDir = CirrusService.isDirectory(b)

      // Folders first unless mixed sorting is enabled
      if (!mixedSorting.value) {
        if (aIsDir && !bIsDir) return -1
        if (!aIsDir && bIsDir) return 1
      }

      return CirrusService.getFileName(a).localeCompare(
        CirrusService.getFileName(b),
        undefined,
        {
          numeric: true,
          sensitivity: 'base',
        },
      )
    })
  }

  return [...props.files].sort((a, b) => {
    const aIsDir = CirrusService.isDirectory(a)
    const bIsDir = CirrusService.isDirectory(b)

    // Folders first unless mixed sorting is enabled
    if (!mixedSorting.value) {
      if (aIsDir && !bIsDir) return -1
      if (!aIsDir && bIsDir) return 1
    }

    let comparison = 0

    if (sortColumn.value === 'name') {
      comparison = CirrusService.getFileName(a).localeCompare(
        CirrusService.getFileName(b),
        undefined,
        {
          numeric: true,
          sensitivity: 'base',
        },
      )
    } else if (sortColumn.value === 'size') {
      comparison = CirrusService.getFileSize(a) - CirrusService.getFileSize(b)
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
  const fileName = CirrusService.getFileName(file)
  if (CirrusService.isDirectory(file)) {
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

.file-table-container--dragging {
  outline: 2px dashed rgba($color-blue-500, 0.35);
}

.file-table-drop-overlay {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: rgba($color-blue-500, 0.12);
  border-radius: inherit;
  transition: opacity 0.2s ease;
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
    background-color: $theme-palette-bg-inverse;

    @media (prefers-color-scheme: dark) {
      background-color: $theme-palette-bg-secondary;
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
  }

  &.file-table-row--selected {
    background-color: $theme-palette-bg-secondary;

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

.file-table-name-container {
  display: flex;
  align-items: center;
  border-radius: $border-radius-md;
  transition: background-color 0.15s ease;
  padding: $spacing-xs $spacing-sm;

  &--drop-target {
    background-color: $color-primary-400;
  }
}

.file-table-name {
  flex: 1;
  margin-left: $spacing-sm;
  color: inherit;
  display: flex;
  align-items: center;
}

.file-table-name-label {
  display: inline-flex;
  align-items: center;
  border-radius: $border-radius-md;
  padding: 0 $spacing-xs;
  transition: background-color 0.15s ease;
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
