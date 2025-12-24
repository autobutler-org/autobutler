<template>
  <div class="file-explorer-folder-controls">
    <button
      id="add-folder-btn"
      class="file-explorer-add-folder btn btn--icon"
      title="Add Folder"
      type="button"
      @click="toggleFolderInput"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="icon icon--base"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
      </svg>
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

<script setup lang="ts">
import { ref, nextTick } from 'vue'

const props = defineProps<{
  currentPath: string
}>()

const emit = defineEmits<{
  'folder-created': [folderName: string]
}>()

const folderInputRef = ref<HTMLInputElement | null>(null)
const showFolderInput = ref(false)
const folderName = ref('')
const isCreating = ref(false)

// TODO: Move folder creation logic to a service/module, then have this function wrap that
const createFolder = async () => {
  if (!folderName.value.trim() || isCreating.value) return

  isCreating.value = true
  try {
    const formData = new FormData()
    formData.append('folderName', folderName.value.trim())

    const folderPath = props.currentPath
      ? `/api/v1/folder/cirrus/${props.currentPath}`
      : '/api/v1/folder/cirrus/'

    const response = await fetch(folderPath, {
      method: 'POST',
      body: formData,
    })

    if (!response.ok) {
      throw new Error('Failed to create folder')
    }

    emit('folder-created', folderName.value.trim())
    folderName.value = ''
    showFolderInput.value = false
  } catch (error) {
    console.error('Error creating folder:', error)
    alert('Failed to create folder')
  } finally {
    isCreating.value = false
  }
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    createFolder()
  } else if (event.key === 'Escape') {
    showFolderInput.value = false
    folderName.value = ''
  }
}

const toggleFolderInput = async () => {
  showFolderInput.value = !showFolderInput.value
  if (showFolderInput.value) {
    await nextTick()
    folderInputRef.value?.focus()
  } else {
    folderName.value = ''
  }
}
</script>

<style lang="scss" scoped>
.file-explorer-add-folder {
  margin-left: 0;
  background: none;
  border: none;
  cursor: pointer;
  padding: $spacing-xs;
  border-radius: $border-radius;

  &:hover {
    background-color: $color-gray-100;
  }

  svg {
    color: $color-primary-400;
    transition: color 0.2s ease;
  }

  &:hover svg {
    color: $color-primary-600;
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
  border: 1px solid $color-gray-300;
  border-radius: $border-radius;
  font-size: $font-size-sm;
  width: 200px;

  &:focus {
    outline: none;
    border-color: $color-primary-500;
    box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
  }

  &:disabled {
    background-color: $color-gray-100;
    cursor: not-allowed;
  }

  @media (prefers-color-scheme: dark) {
    background-color: $color-gray-800;
    border-color: $color-gray-600;
    color: $color-gray-100;

    &:focus {
      border-color: $color-primary-400;
    }

    &:disabled {
      background-color: $color-gray-700;
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
  font-size: $font-size-sm;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  border: 1px solid transparent;

  &--icon {
    padding: $spacing-xs;
  }
}

.icon {
  display: inline-block;
  vertical-align: middle;

  &--base {
    width: 1.25rem;
    height: 1.25rem;
  }
}
</style>
