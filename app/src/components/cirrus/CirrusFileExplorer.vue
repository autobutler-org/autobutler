<template>
  <div id="file-explorer" class="file-explorer">
    <div class="file-explorer-header">
      <div>
        <h2 class="file-explorer-title">Cirrus</h2>
      </div>
      <div style="display: flex; gap: 0.5rem; align-items: center">
        <!-- Navigation and download controls would go here -->
      </div>
    </div>

    <!-- Drop Zone for file uploads -->
    <CirrusDropZone
      :current-path="currentPath"
      @files-uploaded="handleFilesUploaded"
    />

    <div id="file-explorer-selectable">
      <div class="file-explorer-controls">
        <div>
          <CirrusBreadcrumbs
            :current-path="currentPath"
            @navigate="navigateToPath"
            @folder-created="handleFolderCreated"
          />
        </div>
        <div style="display: flex; align-items: center; gap: 1rem">
          <label class="device-badge-toggle">
            <input
              type="checkbox"
              id="toggle-device-badges"
              :checked="showDeviceBadges"
              @change="
                toggleDeviceBadges(($event.target as HTMLInputElement).checked)
              "
            />
            <span>Show device names</span>
          </label>
          <div class="view-switcher">
            <button
              :class="[
                'btn',
                'btn--icon',
                view === 'list' ? 'btn--primary' : 'btn--secondary',
              ]"
              @click="switchView('list')"
              title="List View"
              type="button"
            >
              <ListViewIcon />
            </button>
            <button
              :class="[
                'btn',
                'btn--icon',
                view === 'grid' ? 'btn--primary' : 'btn--secondary',
              ]"
              @click="switchView('grid')"
              title="Grid View"
              type="button"
            >
              <GridViewIcon />
            </button>
          </div>
        </div>
      </div>

      <div id="file-explorer-status" />

      <div id="file-explorer-view-content">
        <template v-if="loading">
          <span class="file-explorer-loading">Loading files...</span>
        </template>
        <template v-else-if="error">
          <span class="file-explorer-error">{{ error }}</span>
        </template>
        <template v-else-if="files.length === 0">
          <span class="file-explorer-empty">No files found</span>
        </template>
        <template v-else-if="view === 'grid'">
          <CirrusGridView
            :files="files"
            :current-path="currentPath"
            :show-device-badges="showDeviceBadges"
            :selected-file="selectedFile"
            @navigate-folder="handleNavigateFolder"
            @open-file="handleOpenFile"
            @select="handleSelectFile"
            @context-menu="handleContextMenu"
          />
        </template>
        <template v-else>
          <CirrusListView
            :files="files"
            :current-path="currentPath"
            :show-device-badges="showDeviceBadges"
            :selected-file="selectedFile"
            @navigate-folder="handleNavigateFolder"
            @open-file="handleOpenFile"
            @select="handleSelectFile"
            @context-menu="handleContextMenu"
          />
        </template>
      </div>
    </div>

    <!-- File Viewer Modal -->
    <CirrusFileViewer
      v-model="fileViewerOpen"
      :file-src="selectedFileSrc"
      :file-type="selectedFileType"
    />

    <!-- Context Menu -->
    <CirrusContextMenu
      v-model="contextMenuOpen"
      :file="contextMenuFile"
      :current-path="currentPath"
      :x="contextMenuX"
      :y="contextMenuY"
      @download="handleDownload"
      @rename="handleRename"
      @details="handleFileDetails"
      @delete="handleDelete"
    />

    <!-- Move/Rename Modal Dialog -->
    <ModalDialog v-if="moveDialogOpen" @close="moveDialogOpen = false">
      <form @submit.prevent="submitMoveDialog" class="move-dialog-form">
        <h3 class="move-dialog-title">Rename or Move</h3>
        <div class="move-dialog-field">
          <label for="move-path-input" class="move-dialog-label"
            >New name or path</label
          >
          <input
            id="move-path-input"
            v-model="moveDialogNewPath"
            :disabled="moveDialogLoading"
            class="move-dialog-input"
            autocomplete="off"
          />
        </div>
        <div v-if="moveDialogError" class="move-dialog-error">
          {{ moveDialogError }}
        </div>
        <div class="move-dialog-actions">
          <button
            type="button"
            class="btn btn--secondary"
            @click="moveDialogOpen = false"
            :disabled="moveDialogLoading"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="btn btn--primary"
            :disabled="moveDialogLoading"
          >
            <span v-if="moveDialogLoading">Moving...</span>
            <span v-else>Move/Rename</span>
          </button>
        </div>
      </form>
    </ModalDialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, onMounted } from 'vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import { moveFile } from '@/services/cirrusService'
import { useRoute, useRouter } from 'vue-router'
import type { CirrusFileNode, FileType } from '@/types/cirrus'
import {
  getFiles,
  determineFileType,
  getFileName,
} from '@/services/cirrusService'
import CirrusBreadcrumbs from './CirrusBreadcrumbs.vue'
import CirrusListView from './CirrusListView.vue'
import CirrusGridView from './CirrusGridView.vue'
import CirrusFileViewer from './CirrusFileViewer.vue'
import CirrusContextMenu from './CirrusContextMenu.vue'
import CirrusDropZone from './CirrusDropZone.vue'
import ListViewIcon from '../icons/ListViewIcon.vue'
import GridViewIcon from '../icons/GridViewIcon.vue'

const route = useRoute()
const router = useRouter()

// State
const currentPath = ref('')
const files = ref<CirrusFileNode[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const view = ref<'list' | 'grid'>('list')
const showDeviceBadges = ref(false)

// File viewer state
const fileViewerOpen = ref(false)
const selectedFileSrc = ref('')
const selectedFileType = ref<FileType>('generic')
// Selection state
const selectedFile = ref<CirrusFileNode | null>(null)

// Context menu state
const contextMenuOpen = ref(false)
const contextMenuFile = ref<CirrusFileNode | null>(null)
const contextMenuX = ref(0)
const contextMenuY = ref(0)

// Move/Rename dialog state
const moveDialogOpen = ref(false)
const moveDialogLoading = ref(false)
const moveDialogError = ref('')
const moveDialogNewPath = ref('')
const moveDialogFile = ref<CirrusFileNode | null>(null)

// Fetch files for the current path
const fetchFiles = async () => {
  loading.value = true
  error.value = null
  try {
    files.value = await getFiles(currentPath.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load files'
    files.value = []
  } finally {
    loading.value = false
  }
}

// Watch route changes to update current path
watch(
  () => {
    return route.params.pathMatch
  },
  (newPath) => {
    if (Array.isArray(newPath)) {
      currentPath.value = newPath.join('/')
    } else {
      currentPath.value = newPath || ''
    }
  },
  { immediate: true },
)

// Watch current path changes to fetch files
watch(currentPath, () => {
  fetchFiles()
})

// Fetch files on mount
onMounted(() => {
  fetchFiles()
})

// Methods
// TODO: Move to a common utility file
const constructFileSrc = (relativePath: string) =>
  `/api/v1/download/cirrus/${relativePath}`

const handleSelectFile = (file: CirrusFileNode) => {
  selectedFile.value = file
}
const navigateToPath = (path: string) => {
  currentPath.value = path
  router.push(`/cirrus${path ? '/' + path : ''}`)
}

const handleNavigateFolder = (path: string) => {
  navigateToPath(path)
}

const handleOpenFile = (file: CirrusFileNode) => {
  // Construct the relative path for the API from currentPath and filename
  const fileName = getFileName(file)
  const relativePath = currentPath.value
    ? `${currentPath.value}/${fileName}`
    : fileName
  selectedFileSrc.value = constructFileSrc(relativePath)
  selectedFileType.value = determineFileType(file)
  fileViewerOpen.value = true
}

const switchView = (newView: 'list' | 'grid') => {
  view.value = newView
}

const toggleDeviceBadges = (show: boolean) => {
  showDeviceBadges.value = show
}

// Handle files uploaded - add them to the list
const handleFilesUploaded = (uploadedFiles: CirrusFileNode[]) => {
  // Add the new files to the list, avoiding duplicates
  for (const newFile of uploadedFiles) {
    const exists = files.value.some((f) => {
      return f.fullPath === newFile.fullPath
    })
    if (!exists) {
      files.value.push(newFile)
    }
  }
}

// Handle folder created - add it to the list
const handleFolderCreated = (folderName: string) => {
  const fullPath = currentPath.value
    ? `${currentPath.value}/${folderName}`
    : folderName
  const exists = files.value.some((f) => {
    return f.fullPath === fullPath
  })
  if (!exists) {
    // Add folder at the beginning (folders typically appear first)
    files.value.unshift({
      name: folderName,
      size: 0,
      isDir: true,
      deviceName: '',
      devicePath: '',
      fullPath,
    })
  }
}

// Context menu handlers
const handleContextMenu = (event: MouseEvent, file: CirrusFileNode) => {
  contextMenuFile.value = file
  contextMenuX.value = event.clientX
  contextMenuY.value = event.clientY
  contextMenuOpen.value = true
}

const handleDownload = (file: CirrusFileNode) => {
  const fileName = getFileName(file)
  const relativePath = currentPath.value
    ? `${currentPath.value}/${fileName}`
    : fileName
  const downloadUrl = `/api/v1/download/cirrus/${relativePath}`

  // Create a temporary link and click it to trigger download
  const link = document.createElement('a')
  link.href = downloadUrl
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

const handleRename = (file: CirrusFileNode) => {
  moveDialogFile.value = file
  moveDialogError.value = ''
  // Default to just renaming the file/folder name, not the whole path
  moveDialogNewPath.value = file.name
  console.log(file)
  moveDialogOpen.value = true
}

const submitMoveDialog = async () => {
  if (!moveDialogFile.value) return
  moveDialogLoading.value = true
  moveDialogError.value = ''
  try {
    const oldPath = moveDialogFile.value.name
    const newPath = moveDialogNewPath.value.trim()
    if (!newPath || newPath === oldPath) {
      moveDialogError.value = 'Please enter a new name or path.'
      moveDialogLoading.value = false
      return
    }
    await moveFile(oldPath, newPath)
    // Update UI: refetch files and close dialog
    await fetchFiles()
    moveDialogOpen.value = false
    moveDialogFile.value = null
  } catch (e) {
    moveDialogError.value =
      e instanceof Error ? e.message : 'Failed to move file.'
  } finally {
    moveDialogLoading.value = false
  }
}

const handleFileDetails = (file: CirrusFileNode) => {
  // TODO: Implement file details dialog
  const fileName = getFileName(file)
  console.log('File details:', fileName)
  alert(
    `File details for: ${fileName}\nPath: ${file.fullPath}\nDevice: ${file.deviceName}`,
  )
}

const handleDelete = async (file: CirrusFileNode) => {
  const fileName = getFileName(file)
  if (!confirm(`Are you sure you want to delete "${fileName}"?`)) {
    return
  }

  try {
    // Build query params - API expects rootDir and filePaths as query parameters
    const params = new URLSearchParams()
    params.append('rootDir', currentPath.value)
    params.append('filePaths', fileName)

    const response = await fetch(`/api/v1/cirrus?${params.toString()}`, {
      method: 'DELETE',
    })

    if (!response.ok) {
      throw new Error('Failed to delete file')
    }

    // Remove the file from the in-memory list
    files.value = files.value.filter((f) => {
      return f.fullPath !== file.fullPath
    })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete file'
  }
}
</script>

<style lang="scss" scoped>
.file-explorer {
  background-color: white;
  max-width: 100%;
  box-shadow: $shadow-sm;
  padding: $spacing-lg;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;

  @media (prefers-color-scheme: dark) {
    background-color: $color-gray-900;
  }
}

.file-explorer-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-md;

  @media (prefers-color-scheme: dark) {
    color: white;
  }
}

.file-explorer-error {
  padding: $spacing-md 0;
  color: #dc2626;

  @media (prefers-color-scheme: dark) {
    color: #f87171;
  }
}

.file-explorer-empty {
  padding: $spacing-md 0;
  color: $color-gray-500;

  @media (prefers-color-scheme: dark) {
    color: white;
  }
}

.file-explorer-header {
  display: flex;
  align-items: center;
  margin-bottom: $spacing-lg;

  @media (prefers-color-scheme: dark) {
    color: white;
  }
}

.file-explorer-loading {
  padding: $spacing-md 0;
  color: $color-gray-500;
  font-style: italic;

  @media (prefers-color-scheme: dark) {
    color: white;
  }
}

#file-explorer-selectable {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.file-explorer-title {
  font-size: $font-size-2xl;
  font-weight: 700;
  margin-right: $spacing-lg;
  white-space: nowrap;
  color: $color-gray-100;

  @media (prefers-color-scheme: dark) {
    color: white;
  }
}

#file-explorer-view-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.device-badge-toggle {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  font-size: $font-size-sm;
  color: $color-gray-600;
  cursor: pointer;

  input {
    cursor: pointer;
  }

  @media (prefers-color-scheme: dark) {
    color: white;
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

  &--primary {
    background-color: $color-primary-600;
    color: white;
    border-color: $color-primary-600;

    &:hover {
      background-color: $color-primary-400;
    }
  }

  &--secondary {
    background-color: transparent;
    color: $color-gray-700;
    border-color: $color-gray-300;

    &:hover {
      background-color: $color-gray-100;
    }

    @media (prefers-color-scheme: dark) {
      color: $color-gray-300;
      border-color: $color-gray-600;

      &:hover {
        background-color: $color-gray-800;
      }
    }
  }
}

.move-dialog-actions {
  display: flex;
  gap: $spacing-md;
  justify-content: flex-end;
  margin-top: $spacing-md;
}

.move-dialog-error {
  color: #dc2626;
  font-size: $font-size-sm;
  margin-bottom: $spacing-md;
  text-align: left;
}

.move-dialog-field {
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
  margin-bottom: $spacing-md;
}

.move-dialog-form {
  min-width: 540px;
  max-width: 98vw;
  background: $color-gray-800;
  border-radius: $border-radius-lg;
  box-shadow: $shadow-lg;
  padding: $spacing-xl $spacing-lg $spacing-lg $spacing-lg;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: $spacing-lg;

  button.btn {
    min-width: 110px;
    font-size: $font-size-base;
    font-weight: 600;
    border-radius: $border-radius-md;
    padding: $spacing-sm $spacing-lg;
  }

  @media (max-width: 480px) {
    min-width: 0;
    padding: $spacing-lg;
  }
}

.move-dialog-input {
  padding: $spacing-md;
  border: 1.5px solid $color-gray-300;
  border-radius: $border-radius-md;
  font-size: $font-size-lg;
  width: 100%;
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
  background: $color-gray-50;
  color: $color-gray-100;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.01);

  &:focus {
    outline: none;
    border-color: $color-primary-500;
    box-shadow: 0 0 0 2px $color-primary-100;
  }

  @media (prefers-color-scheme: dark) {
    background: $color-gray-900;
    color: $color-gray-100;
    border-color: $color-gray-600;
    &::placeholder {
      color: $color-gray-400;
      opacity: 1;
    }
  }
}

.move-dialog-label {
  font-size: $font-size-base;
  font-weight: 500;
  color: $color-gray-200;
  margin-bottom: $spacing-xs;
}

.move-dialog-title {
  font-size: $font-size-xl;
  font-weight: 700;
  margin-bottom: $spacing-md;
  text-align: left;
  color: $color-gray-100;
}

.view-switcher {
  display: flex;
  gap: $spacing-xs;
}
</style>
