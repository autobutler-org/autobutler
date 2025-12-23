<template>
  <nav class="file-explorer-breadcrumbs" :data-path="currentPath">
    <span v-for="(segment, index) in segments" :key="index" class="file-explorer-breadcrumb">
      <a href="#" @click.prevent="navigateTo(segment.path)">
        {{ segment.name }}
      </a>
      <span>/</span>
    </span>
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
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 4v16m8-8H4"
          />
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
  </nav>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { useRouter } from 'vue-router'

const props = defineProps<{
  currentPath: string
}>()

const emit = defineEmits<{
  navigate: [path: string]
  'folder-created': [folderName: string]
}>()

const router = useRouter()

// Folder input state
const showFolderInput = ref(false)
const folderName = ref('')
const folderInputRef = ref<HTMLInputElement | null>(null)
const isCreating = ref(false)

// Parse the path into breadcrumb segments
const segments = computed(() => {
  const parts = props.currentPath.split('/').filter((p) => {
    return p.length > 0
  })
  const result = [{ name: 'cirrus', path: '' }]

  let accumulatedPath = ''
  for (const part of parts) {
    accumulatedPath = accumulatedPath ? `${accumulatedPath}/${part}` : part
    result.push({ name: part, path: accumulatedPath })
  }

  return result
})

const navigateTo = (path: string) => {
  emit('navigate', path)
  router.push(`/cirrus${path ? '/' + path : ''}`)
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
</script>

<style lang="scss" scoped>
.file-explorer-breadcrumbs {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  font-size: $font-size-sm;
  color: $color-gray-500;
  margin-bottom: $spacing-sm;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-400;
  }
}

.file-explorer-breadcrumb {
  color: $color-gray-700;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-300;
  }

  a {
    color: inherit;
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }
}

.file-explorer-folder-controls {
  display: inline-flex;
  align-items: center;
  margin-left: $spacing-sm;
  gap: $spacing-sm;
}

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

.icon {
  display: inline-block;
  vertical-align: middle;

  &--base {
    width: 1.25rem;
    height: 1.25rem;
  }
}

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
</style>
