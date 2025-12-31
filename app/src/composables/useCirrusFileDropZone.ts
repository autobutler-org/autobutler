import { ref, unref, type Ref } from 'vue'

import CirrusService from '@/services/cirrusService'
import type { CirrusFileNode } from '@/types/cirrus'

type MaybeRef<T> = T | Ref<T>

interface UseCirrusFileDropZoneOptions {
  currentPath: MaybeRef<string>
  onFilesUploaded: (files: CirrusFileNode[]) => void
}

export const useCirrusFileDropZone = ({
  currentPath,
  onFilesUploaded,
}: UseCirrusFileDropZoneOptions) => {
  const isDragOver = ref(false)
  const isUploading = ref(false)
  const uploadProgress = ref('')
  const fileInputRef = ref<HTMLInputElement | null>(null)

  const resolveCurrentPath = () => (unref(currentPath) || '').trim()

  const resolveUploadPath = () => {
    const path = resolveCurrentPath()
    return path ? `/api/v1/cirrus/${path}` : '/api/v1/cirrus'
  }

  const uploadFiles = async (files: FileList) => {
    if (!files.length) return

    isUploading.value = true
    uploadProgress.value = `Uploading ${files.length} file${files.length > 1 ? 's' : ''}...`

    try {
      await CirrusService.uploadFiles(resolveUploadPath(), files)

      const currentPathValue = resolveCurrentPath()

      const uploadedNodes: CirrusFileNode[] = Array.from(files).map((file) => ({
        name: file.name,
        size: file.size,
        isDir: false,
        deviceName: '',
        devicePath: '',
        fullPath: currentPathValue ? `${currentPathValue}/${file.name}` : file.name,
      }))

      uploadProgress.value = 'Upload complete!'
      onFilesUploaded(uploadedNodes)

      setTimeout(() => {
        uploadProgress.value = ''
      }, 2000)
    } catch (error) {
      uploadProgress.value = `Upload failed: ${
        error instanceof Error ? error.message : 'Unknown error'
      }`

      setTimeout(() => {
        uploadProgress.value = ''
      }, 3000)
    } finally {
      isUploading.value = false
    }
  }

  const handleDragEnter = (event: DragEvent) => {
    event.preventDefault()
    isDragOver.value = true
  }

  const handleDragOver = (event: DragEvent) => {
    event.preventDefault()
    isDragOver.value = true
  }

  const handleDragLeave = (event: DragEvent) => {
    event.preventDefault()

    const currentTarget = event.currentTarget as Node | null
    const relatedTarget = event.relatedTarget as Node | null

    // Ignore leaves that move within the current drop zone
    if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget)) {
      return
    }

    isDragOver.value = false
  }

  const handleDrop = async (event: DragEvent) => {
    event.preventDefault()
    isDragOver.value = false

    const files = event.dataTransfer?.files
    if (files && files.length) {
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
      input.value = ''
    }
  }

  return {
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
  }
}

export type UseCirrusFileDropZoneReturn = ReturnType<typeof useCirrusFileDropZone>
