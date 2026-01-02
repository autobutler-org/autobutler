<template>
  <div class="file-explorer-folder-controls">
    <button
      id="add-folder-btn"
      class="file-explorer-add-folder btn btn--icon"
      title="Add Folder"
      type="button"
      @click="toggleFolderInput"
    >
      <AddFolderIcon />
    </button>
    <input
      v-if="showFolderInput"
      ref="folderInputRef"
      v-model="folderName"
      type="text"
      class="file-explorer-folder-input"
      placeholder="New folder name"
      maxlength="255"
      :disabled="isCreating"
      @keydown="handleKeydown"
      @blur="showFolderInput = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { nextTick, ref } from 'vue';

import FolderService from '@/services/folderService';
import AddFolderIcon from '../icons/AddFolderIcon.vue';

const props = defineProps<{
  currentPath: string;
}>();

const emit = defineEmits<{
  'folder-created': [folderName: string];
}>();

const folderInputRef = ref<HTMLInputElement | null>(null);
const showFolderInput = ref(false);
const folderName = ref('');
const isCreating = ref(false);

// TODO: Move folder creation logic to a service/module, then have this function wrap that
const createFolder = async () => {
  if (!folderName.value.trim() || isCreating.value) return;

  isCreating.value = true;
  try {
    await FolderService.createFolder(
      props.currentPath,
      folderName.value.trim(),
    );
    emit('folder-created', folderName.value.trim());
    folderName.value = '';
    showFolderInput.value = false;
  } catch (error) {
    console.error('Error creating folder:', error);
    alert('Failed to create folder');
  } finally {
    isCreating.value = false;
  }
};

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    createFolder();
  } else if (event.key === 'Escape') {
    showFolderInput.value = false;
    folderName.value = '';
  }
};

const toggleFolderInput = async () => {
  showFolderInput.value = !showFolderInput.value;
  if (showFolderInput.value) {
    await nextTick();
    folderInputRef.value?.focus();
  } else {
    folderName.value = '';
  }
};
</script>

<style lang="scss" scoped>
.file-explorer-add-folder {
  margin-left: 0;
  background: none;
  border: 1px solid $theme-palette-border;
  cursor: pointer;
  padding: $spacing-xs;
  border-radius: $border-radius;

  &:hover {
    background: rgba($theme-palette-accent, 0.12);
    border-color: $theme-palette-accent;
  }

  svg {
    color: $theme-palette-accent;
    transition: color 0.2s ease;
  }

  &:hover svg {
    color: $theme-palette-accent-hover;
  }
}

.file-explorer-folder-controls {
  display: inline-flex;
  align-items: center;
  margin-left: $spacing-sm;
  gap: $spacing-sm;
}

.file-explorer-folder-input {
  padding: $spacing-xs $spacing-sm;
  border: 1px solid $theme-palette-border;
  border-radius: $border-radius;
  font-size: $theme-font-size-sm;
  width: 200px;

  &:focus {
    outline: none;
    border-color: $theme-palette-accent;
    box-shadow: 0 0 0 2px rgba($theme-palette-accent, 0.2);
  }

  &:disabled {
    background-color: $theme-palette-bg-secondary;
    cursor: not-allowed;
  }

  @media (prefers-color-scheme: dark) {
    background-color: $theme-palette-bg-primary;
    border-color: $theme-palette-border;
    color: $theme-palette-text-primary;

    &:focus {
      border-color: $theme-palette-accent;
    }

    &:disabled {
      background-color: $theme-palette-bg-secondary;
    }
  }
}

// Global overrides
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius;
  font-size: $theme-font-size-sm;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  border: 1px solid transparent;

  &--icon {
    padding: $spacing-xs;
  }
}
</style>
