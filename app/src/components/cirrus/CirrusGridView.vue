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
            </div>
            <DeviceBadge
              v-if="props.showDeviceBadges && file.deviceName"
              :device-name="file.deviceName"
            />
            <div v-if="!CirrusService.isDirectory(file)" class="grid-view-size">
              {{ CirrusService.formatBytes(CirrusService.getFileSize(file)) }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
  <div
    class="grid-view-deselect-area"
    @click="
      console.log('deselect grid');
      emit('deselect-all');
    "
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
import type { CirrusFileNode } from '@/types/cirrus';
import { computed, ref, watch } from 'vue';

const props = defineProps<{
  files: CirrusFileNode[];
  currentPath: string;
  showDeviceBadges?: boolean;
  selectedFiles?: CirrusFileNode[];
}>();

const emit = defineEmits<{
  'navigate-folder': [path: string];
  'open-file': [file: CirrusFileNode];
  'context-menu': [event: MouseEvent, file: CirrusFileNode];
  select: [file: CirrusFileNode, event?: MouseEvent];
  'files-uploaded': [files: CirrusFileNode[]];
  'deselect-all': [];
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

const hoveredDirectoryPath = ref<string | null>(null);

const normalizeCurrentPath = computed(() =>
  CirrusService.normalizePath(props.currentPath),
);

const resolveDirectoryTargetPath = (file: CirrusFileNode) => {
  const directoryName = CirrusService.getFileName(file);
  const basePath = normalizeCurrentPath.value;
  return basePath ? `${basePath}/${directoryName}` : directoryName;
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
  hoveredDirectoryPath.value = resolveDirectoryTargetPath(file);
};

const handleDirectoryDragOver = (event: DragEvent, file: CirrusFileNode) => {
  if (!CirrusService.isDirectory(file)) return;
  event.preventDefault();
  hoveredDirectoryPath.value = resolveDirectoryTargetPath(file);
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
      ? `${props.currentPath}/${fileName}`
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
