<template>
  <div
    class="grid-view-container"
    :class="{ 'grid-view-container--dragging': isDragOver }"
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
  >
    <div class="grid-view-drop-overlay" v-show="isDragOver" />
    <div class="grid-view-grid">
      <div
        v-for="file in files"
        :key="file.fullPath"
        :class="[
          'grid-view-item',
          'file-node',
          {
            'grid-view-item--folder': CirrusService.isDirectory(file),
            'grid-view-item--selected':
              props.selectedFiles &&
              props.selectedFiles.some((f) => f.fullPath === file.fullPath),
            'grid-view-item--drop-target':
              CirrusService.isDirectory(file) &&
              hoveredDirectoryPath === resolveDirectoryTargetPath(file),
          },
        ]"
        :data-name="CirrusService.getFileName(file)"
        :data-is-folder="CirrusService.isDirectory(file)"
        :data-file-type="CirrusService.determineFileType(file)"
        :data-device-name="file.deviceName"
        @click="(event) => emit('select', file, event)"
        @dblclick="handleClick(file)"
        @contextmenu="handleContextMenu($event, file)"
        @dragenter="handleDirectoryDragEnter($event, file)"
        @dragover="handleDirectoryDragOver($event, file)"
        @dragleave="handleDirectoryDragLeave($event, file)"
        @drop="handleDirectoryDrop($event, file)"
        @dragstart="handleFileDragStart($event, file)"
        draggable="true"
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
          <div
            v-if="CirrusService.isDirectory(file)"
            class="grid-view-icon-container"
          >
            <CirrusFolderIcon />
          </div>
          <div v-else class="grid-view-icon-container">
            <component
              :is="getIconComponent(CirrusService.determineFileType(file))"
            />
          </div>
          <div class="grid-view-details">
            <div
              class="grid-view-name"
              :title="CirrusService.getFileName(file)"
            >
              {{ CirrusService.getFileName(file) }}
              <span
                v-if="file.isBackedUp === false && !file.isDir"
                class="backup-dot"
                title="Not backed up"
              ></span>
            </div>
            <DeviceBadge
              v-if="props.showDeviceBadges && file.deviceName"
              :device-name="file.deviceName"
              @click.stop="emit('rename', file)"
            />
            <div
              v-if="
                props.showFileSizes !== false &&
                !CirrusService.isDirectory(file)
              "
              class="grid-view-size"
            >
              {{ CirrusService.formatBytes(CirrusService.getFileSize(file)) }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
  <div
    class="grid-view-deselect-area"
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
import { useCirrusFileDropZone } from '@/composables/useCirrusFileDropZone';
import CirrusService from '@/services/cirrusService';
import type { CirrusDragFileData, CirrusFileNode } from '@/types/cirrus';
import { joinPathsNormalized, normalizePath } from '@/util/filepath';
import { computed, ref, watch } from 'vue';

const props = defineProps<{
  files: CirrusFileNode[];
  currentPath: string;
  showDeviceBadges?: boolean;
  showFileSizes?: boolean;
  selectedFiles?: CirrusFileNode[];
  // Parent-controlled sorting (optional for grid)
  sortColumn: 'name' | 'size' | 'device' | null;
  sortDirection: 'asc' | 'desc';
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
  'file-move': [
    moves: Array<{
      oldPath: string;
      newPath: string;
      oldDeviceSerial?: string;
      newDeviceSerial?: string;
    }>,
  ];
}>();

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
      const files = JSON.parse(multi) as CirrusDragFileData[];
      const moves = files
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
  // Otherwise, treat as upload
  await handleDrop(event, targetPath);
};

// TODO: CirrusListView has the exact same functions/code, after this point

// TODO: Move this to a utility module
const getIconComponent = (fileType: string) => {
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
.grid-view-container {
  overflow-y: auto;
  padding: $spacing-sm;
  position: relative;
  border-radius: $border-radius-lg;
}

.grid-view-container--dragging {
  outline: 2px dashed rgba($color-blue-500, 0.35);
}

.grid-view-drop-overlay {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: rgba($color-blue-500, 0.12);
  border-radius: inherit;
  transition: opacity 0.2s ease;
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
    background-color: $theme-palette-bg-secondary;
    .context-menu-trigger {
      opacity: 1;
    }
  }
  &.grid-view-item--selected {
    background-color: $theme-palette-accent;
  }
  &.grid-view-item--drop-target {
    background-color: rgba($color-blue-500, 0.2);
    outline: 2px dashed rgba($color-blue-500, 0.45);
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
  font-size: $theme-font-size-base;
  color: $theme-palette-text-muted;
  border-radius: $border-radius-sm;
  opacity: 0;
  transition:
    opacity 0.15s ease,
    background-color 0.15s ease;

  &:hover {
    background-color: $theme-palette-accent-hover;
    color: $theme-palette-text-inverse;
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
  font-size: $theme-font-size-sm;
  word-break: break-word;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;

  &::selection {
    background-color: transparent;
  }
}

.backup-dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #ff6b6b;
  margin-left: 8px;
  vertical-align: middle;
}

.grid-view-size {
  font-size: 0.75rem;
  color: $theme-palette-text-muted;
  margin-top: $spacing-xs;
}

.grid-view-deselect-area {
  flex: 1;
}
</style>
