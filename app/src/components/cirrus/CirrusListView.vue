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
            v-for="column in displayedColumns"
            :key="column.column ? column.column : ''"
            :header="column.column"
            :active-sort-column="props.sortColumn"
            :sort-direction="props.sortDirection"
            :align-direction="column.alignDirection"
            @toggle:sort="toggleSort"
          />
          <th class="file-table-header-cell file-table-header-cell--toggle">
            <button
              class="sort-switcher"
              type="button"
              :title="
                props.mixedSorting
                  ? 'Switch to Folders First sorting'
                  : 'Switch to Mixed sorting'
              "
              @click="toggleMixedSorting"
            >
              <SortSwitcherIcon :mixed-sorting="props.mixedSorting" />
            </button>
          </th>
        </tr>
      </thead>
      <tbody id="file-explorer-list" class="file-table-body">
        <tr
          v-for="file in props.files"
          :key="`${file.fullPath}-${file.deviceSerial}`"
          class="file-table-row file-node"
          :class="{
            'file-table-row--selected':
              props.selectedFiles &&
              props.selectedFiles.some((f) => f.fullPath === file.fullPath),
          }"
          :data-name="CirrusService.getFileName(file)"
          :data-file-type="CirrusService.determineFileType(file)"
          :data-device-name="file.deviceName"
          @click="(event) => emit('select', file, event)"
          @dblclick="handleClick(file)"
          @contextmenu="handleContextMenu($event, file)"
          @dragenter="handleDirectoryDragEnter($event, file)"
          @dragover="handleDirectoryDragOver($event, file)"
          @dragleave="handleDirectoryDragLeave($event, file)"
          @drop="handleDirectoryDrop($event, file)"
          draggable="true"
          @dragstart="handleFileDragStart($event, file)"
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
          </td>
          <td
            v-if="props.showDeviceBadges"
            class="file-table-cell file-table-device"
          >
            <DeviceBadge
              v-if="props.showDeviceBadges && file.deviceName"
              :device-name="file.deviceName"
              @click.stop="emit('rename', file)"
            />
          </td>
          <td
            v-if="props.showFileSizes !== false"
            class="file-table-cell file-table-size"
          >
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
    <!-- Deselect area directly after #file-explorer-table -->
  </div>
  <div
    class="file-explorer-deselect-area"
    @click="emit('deselect-all')"
    title="Click to deselect all"
  ></div>
</template>

<script lang="ts" setup>
import DeviceBadge from '@/components/badges/DeviceBadge.vue';
import ArchiveIcon from '@/components/icons/ArchiveIcon.vue';
import CirrusFolderIcon from '@/components/icons/CirrusFolderIcon.vue';
import DocxIcon from '@/components/icons/DocxIcon.vue';
import GenericIcon from '@/components/icons/GenericIcon.vue';
import ImageIcon from '@/components/icons/ImageIcon.vue';
import PdfIcon from '@/components/icons/PdfIcon.vue';
import SlideshowIcon from '@/components/icons/SlideshowIcon.vue';
import SortSwitcherIcon from '@/components/icons/SortSwitcherIcon.vue';
import { useCirrusFileDropZone } from '@/composables/useCirrusFileDropZone';
import CirrusService from '@/services/cirrusService';
import type { CirrusFileNode } from '@/types/cirrus';
import { joinPathsNormalized, normalizePath } from '@/util/filepath';
import { computed, ref, watch, type Component } from 'vue';
import CirrusListViewSortHeader, {
  type HeaderAlignDirection,
  type SortColumn,
  type SortDirection,
} from './CirrusListViewSortHeader.vue';

const props = defineProps<{
  files: CirrusFileNode[];
  currentPath: string;
  showDeviceBadges?: boolean;
  showFileSizes?: boolean;
  selectedFiles?: CirrusFileNode[];
  // Parent-controlled sorting
  sortColumn: SortColumn;
  sortDirection: SortDirection;
  mixedSorting: boolean;
}>();

const emit = defineEmits<{
  'navigate-folder': [path: string];
  'open-file': [file: CirrusFileNode];
  'context-menu': [event: MouseEvent, file: CirrusFileNode];
  select: [file: CirrusFileNode, event?: MouseEvent];
  rename: [file: CirrusFileNode];
  'files-uploaded': [files: CirrusFileNode[]];
  'deselect-all': [];
  'request-sort': [column: SortColumn];
  'toggle-mixed-sorting': [];
  'file-move': [
    moves: Array<{
      oldPath: string;
      newPath: string;
      oldDeviceSerial?: string;
      newDeviceSerial?: string;
    }>,
  ];
}>();

const handleFileDragStart = (event: DragEvent, file: CirrusFileNode) => {
  // Multi-file drag support
  const selected =
    props.selectedFiles &&
    props.selectedFiles.length > 1 &&
    props.selectedFiles.some((f) => f.fullPath === file.fullPath)
      ? props.selectedFiles
      : [file];
  if (selected.length > 1) {
    const payload = selected.map((f) => ({
      path: normalizePath(
        props.currentPath
          ? joinPathsNormalized(props.currentPath, CirrusService.getFileName(f))
          : CirrusService.getFileName(f),
      ),
      deviceSerial: f.deviceSerial || undefined,
    }));
    event.dataTransfer?.setData(
      'application/x-cirrus-multi',
      JSON.stringify(payload),
    );
  } else {
    const filePath = normalizePath(
      props.currentPath
        ? joinPathsNormalized(
            props.currentPath,
            CirrusService.getFileName(file),
          )
        : CirrusService.getFileName(file),
    );
    event.dataTransfer?.setData('application/x-cirrus-file-path', filePath);
    if (file.deviceSerial) {
      event.dataTransfer?.setData(
        'application/x-cirrus-device-serial',
        file.deviceSerial,
      );
    }
  }
};

const sortColumns: {
  column: SortColumn;
  alignDirection?: HeaderAlignDirection;
}[] = [
  {
    column: 'name',
    alignDirection: 'left',
  },
  { column: 'device' },
  { column: 'size' },
];

const sortColumnsNoDevice = sortColumns.filter(
  (col) => col.column !== 'device',
);

// Compute which columns to display based on props
const displayedColumns = computed(() => {
  let cols = props.showDeviceBadges ? sortColumns : sortColumnsNoDevice;
  if (props.showFileSizes === false) {
    cols = cols.filter((c) => c.column !== 'size');
  }
  return cols;
});

const {
  isDragOver,
  handleDragEnter,
  handleDragOver,
  handleDragLeave,
  handleDrop,
} = useCirrusFileDropZone({
  currentPath: computed(() => props.currentPath),
  onFilesUploaded: (files) => emit('files-uploaded', files),
});

const hoveredDirectoryPath = ref<string | null>(null);

const normalizeCurrentPath = computed(() => normalizePath(props.currentPath));

const resolveDirectoryTargetPath = (file: CirrusFileNode) => {
  const directoryName = CirrusService.getFileName(file);
  const basePath = normalizeCurrentPath.value;
  return basePath
    ? joinPathsNormalized(basePath, directoryName)
    : directoryName;
};

const clearHoveredDirectory = () => {
  hoveredDirectoryPath.value = null;
};

watch(isDragOver, (active) => {
  if (!active) {
    clearHoveredDirectory();
  }
});

const handleDirectoryDragEnter = (event: DragEvent, file: CirrusFileNode) => {
  if (!CirrusService.isDirectory(file)) return;
  event.preventDefault();
  const targetPath = resolveDirectoryTargetPath(file);
  // Get the dragged file path from dataTransfer
  const movedFilePath = event.dataTransfer?.getData(
    'application/x-cirrus-file-path',
  );
  // Don't highlight if dragging a folder into itself or its subdirectory
  if (movedFilePath && isSubPath(movedFilePath, targetPath)) {
    return;
  }
  hoveredDirectoryPath.value = targetPath;
};

const handleDirectoryDragOver = (event: DragEvent, file: CirrusFileNode) => {
  if (!CirrusService.isDirectory(file)) return;
  event.preventDefault();
  const targetPath = resolveDirectoryTargetPath(file);
  // Get the dragged file path from dataTransfer
  const movedFilePath = event.dataTransfer?.getData(
    'application/x-cirrus-file-path',
  );
  // Don't highlight if dragging a folder into itself or its subdirectory
  if (movedFilePath && isSubPath(movedFilePath, targetPath)) {
    return;
  }
  hoveredDirectoryPath.value = targetPath;
};

const handleDirectoryDragLeave = (event: DragEvent, file: CirrusFileNode) => {
  if (!CirrusService.isDirectory(file)) return;
  event.preventDefault();

  const currentTarget = event.currentTarget as Node | null;
  const relatedTarget = event.relatedTarget as Node | null;

  if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget)) {
    return;
  }

  if (hoveredDirectoryPath.value === resolveDirectoryTargetPath(file)) {
    clearHoveredDirectory();
  }
};

const handleDirectoryDrop = async (event: DragEvent, file: CirrusFileNode) => {
  if (!CirrusService.isDirectory(file)) return;
  event.preventDefault();
  event.stopPropagation();
  const targetPath = resolveDirectoryTargetPath(file);
  clearHoveredDirectory();

  const dt = event.dataTransfer;
  const multi = dt?.getData('application/x-cirrus-multi');
  if (multi) {
    try {
      const files = JSON.parse(multi);
      const moves = (files as Array<{ path: string; deviceSerial?: string }>)
        .filter((f) => !isSubPath(f.path, targetPath))
        .map((f) => {
          const fileName = f.path.split('/').pop();
          const cleanTargetPath = normalizePath(targetPath);
          const newPath = joinPathsNormalized(cleanTargetPath, fileName || '');
          return {
            oldPath: f.path,
            newPath,
            oldDeviceSerial: f.deviceSerial,
            newDeviceSerial: file.deviceSerial,
          };
        });
      if (moves.length > 0) emit('file-move', moves);
    } catch {}
    return;
  }
  // Single file fallback
  const movedFilePath = dt?.getData('application/x-cirrus-file-path');
  const movedDeviceSerial =
    dt?.getData('application/x-cirrus-device-serial') || undefined;
  if (movedFilePath) {
    if (isSubPath(movedFilePath, targetPath)) return;
    const fileName = movedFilePath.split('/').pop();
    const cleanTargetPath = normalizePath(targetPath);
    const newPath = joinPathsNormalized(cleanTargetPath, fileName || '');
    emit('file-move', [
      {
        oldPath: movedFilePath,
        newPath,
        oldDeviceSerial: movedDeviceSerial,
        newDeviceSerial: file.deviceSerial,
      },
    ]);
    return;
  }
};

const toggleSort = (column: SortColumn): void => {
  emit('request-sort', column);
};

const toggleMixedSorting = () => {
  emit('toggle-mixed-sorting');
};

// TODO: CirrusGridView has the exact same functions/code, after this point

// TODO: Move this to a utility module
const getIconComponent = (fileType: string): Component => {
  switch (fileType) {
    case 'folder':
      return CirrusFolderIcon;
    case 'pdf':
      return PdfIcon;
    case 'image':
      return ImageIcon;
    case 'slideshow':
      return SlideshowIcon;
    case 'archive':
      return ArchiveIcon;
    case 'docx':
      return DocxIcon;
    default:
      return GenericIcon;
  }
};

const handleClick = (file: CirrusFileNode) => {
  const fileName = CirrusService.getFileName(file);
  if (CirrusService.isDirectory(file)) {
    // Navigate to folder
    const newPath = props.currentPath
      ? joinPathsNormalized(props.currentPath, fileName)
      : fileName;
    emit('navigate-folder', newPath);
  } else {
    emit('open-file', file);
  }
};

const handleContextMenu = (event: MouseEvent, file: CirrusFileNode) => {
  event.preventDefault();
  emit('context-menu', event, file);
};

// Utility to check if a path is a subpath of another
const isSubPath = (parent: string, child: string) => {
  const normParent = normalizePath(parent) + '/';
  const normChild = normalizePath(child) + '/';
  return normChild.startsWith(normParent);
};
</script>

<style lang="scss" scoped>
.file-explorer-deselect-area {
  flex: 1;
}

.file-table-container {
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
  display: block;
  align-items: center;
  border-radius: $border-radius-md;
  padding: 0 $spacing-xs;
  transition: background-color 0.15s ease;
  white-space: normal;
  overflow-wrap: anywhere;
  word-break: break-word;

  &::selection {
    background-color: transparent;
  }
}

.file-table-device {
  color: $theme-palette-text-muted;
  font-size: $theme-font-size-sm;
  text-align: right;
  padding-right: $spacing-sm;
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
