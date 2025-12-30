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
import type { CirrusFileNode } from '@/types/cirrus'
import { ref } from 'vue'
import UploadIcon from '../icons/UploadIcon.vue'

const props = defineProps<{
  currentPath: string
}>()

const emit = defineEmits<{
  'files-uploaded': [files: CirrusFileNode[]]
}>()

const isDragOver = ref(false)
const isUploading = ref(false)
const uploadProgress = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)

const handleDragEnter = (event: DragEvent) => {
  event.preventDefault()
  isDragOver.value = true
}

const handleDragOver = (event: DragEvent) => {
  event.preventDefault()
}

const handleDragLeave = (event: DragEvent) => {
  event.preventDefault()
  isDragOver.value = false
}

const handleDrop = async (event: DragEvent) => {
  event.preventDefault()
  isDragOver.value = false

  const files = event.dataTransfer?.files
  if (files && files.length > 0) {
    await uploadFiles(files)
  }
}

const handleClick = () => {
  fileInputRef.value?.click()
}

const handleFileInputChange = (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    uploadFiles(input.files)
    // Reset the input so the same file can be selected again
    input.value = ''
  }
}

// TODO: Move the core upload logic to it's own service/module, then have this function wrap that
const uploadFiles = async (files: FileList) => {
  isUploading.value = true
  uploadProgress.value = `Uploading ${files.length} file${files.length > 1 ? 's' : ''}...`

  try {
    const formData = new FormData()
    for (const file of files) {
      formData.append('files', file)
    }

    // Build the upload URL
    const uploadPath = props.currentPath
      ? `/api/v1/cirrus/${props.currentPath}`
      : '/api/v1/cirrus'

    const response = await fetch(uploadPath, {
      method: 'POST',
      body: formData,
    })

    if (!response.ok) {
      throw new Error('Upload failed')
    }

    // Create file nodes from the uploaded files
    const uploadedNodes: CirrusFileNode[] = Array.from(files).map((file) => {
      return {
        name: file.name,
        size: file.size,
        isDir: false,
        deviceName: '',
        devicePath: '',
        fullPath: props.currentPath
          ? `${props.currentPath}/${file.name}`
          : file.name,
      }
    })

    uploadProgress.value = 'Upload complete!'
    emit('files-uploaded', uploadedNodes)

    // Clear the message after a short delay
    setTimeout(() => {
      uploadProgress.value = ''
    }, 2000)
  } catch (error) {
    uploadProgress.value = `Upload failed: ${error instanceof Error ? error.message : 'Unknown error'}`
    setTimeout(() => {
      uploadProgress.value = ''
    }, 3000)
  } finally {
    isUploading.value = false
  }
}
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
    background-color: darken($theme-palette-bg-secondary, 2%);
  }

  &--active {
    border-color: $theme-palette-accent;
    background-color: lighten($theme-palette-bg-secondary, 4%);
    border-style: solid;
  }

  &--uploading {
    cursor: wait;
    border-color: $theme-palette-success;
    background-color: lighten($theme-palette-bg-secondary, 8%);
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
