<template>
  <div
    :class="[
      'drop-zone',
      { 'drop-zone--active': isDragOver, 'drop-zone--uploading': isUploading },
    ]"
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
    @click="handleClick"
  >
    <input
      ref="fileInputRef"
      type="file"
      class="drop-zone-input"
      multiple
      @change="handleFileInputChange"
    />
    <div class="drop-zone-content">
      <template v-if="isUploading || uploadProgress">
        <span class="drop-zone-text">{{ uploadProgress }}</span>
      </template>
      <template v-else-if="isDragOver">
        <span class="drop-zone-text">Drop files here...</span>
      </template>
      <template v-else>
        <UploadIcon />
        <span class="drop-zone-text">Drop files here or click to upload</span>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

import { useCirrusFileDropZone } from '@/composables/useCirrusFileDropZone'
import type { CirrusFileNode } from '@/types/cirrus'
import UploadIcon from '../icons/UploadIcon.vue'

const props = defineProps<{
  currentPath: string
}>()

const emit = defineEmits<{
  'files-uploaded': [files: CirrusFileNode[]]
}>()

const {
  fileInputRef,
  isDragOver,
  isUploading,
  uploadProgress,
  handleDragEnter,
  handleDragOver,
  handleDragLeave,
  handleDrop,
  handleClick,
  handleFileInputChange,
} = useCirrusFileDropZone({
  currentPath: computed(() => props.currentPath),
  onFilesUploaded: (files) => emit('files-uploaded', files),
})
</script>

<style lang="scss" scoped>
.drop-zone {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: $spacing-md;
  border: 2px dashed $theme-palette-border;
  border-radius: $border-radius-lg;
  background-color: $theme-palette-bg-secondary;
  cursor: pointer;
  transition: all 0.2s ease;
  min-height: 80px;

  &:hover {
    border-color: $theme-palette-accent;
    background-color: hsl(from $theme-palette-bg-secondary h s calc(l - 2));
  }

  &--active {
    border-color: $theme-palette-accent;
    background-color: hsl(from $theme-palette-bg-secondary h s calc(l + 4));
    border-style: solid;
  }

  &--uploading {
    cursor: wait;
    border-color: $theme-palette-success;
    background-color: hsl(from $theme-palette-bg-secondary h s calc(l + 8));
  }
}

.drop-zone-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-sm;
  pointer-events: none;
}

.drop-zone-input {
  display: none;
}

.drop-zone-text {
  font-size: $theme-font-size-sm;
  color: $theme-palette-text-muted;
}
</style>
