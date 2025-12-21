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
    <CirrusDropZone :current-path="currentPath" @files-uploaded="handleFilesUploaded" />

    <div id="file-explorer-selectable">
      <div class="file-explorer-controls">
        <div>
          <CirrusBreadcrumb
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
              @change="toggleDeviceBadges(($event.target as HTMLInputElement).checked)"
            />
            <span>Show device names</span>
          </label>
          <div class="view-switcher">
            <button
              :class="['btn', 'btn--icon', view === 'list' ? 'btn--primary' : 'btn--secondary']"
              @click="switchView('list')"
              title="List View"
              type="button"
            >
              <!-- List view icon -->
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
                  d="M4 6h16M4 12h16M4 18h16"
                />
              </svg>
            </button>
            <button
              :class="['btn', 'btn--icon', view === 'grid' ? 'btn--primary' : 'btn--secondary']"
              @click="switchView('grid')"
              title="Grid View"
              type="button"
            >
              <!-- Grid view icon -->
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
                  d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"
                />
              </svg>
            </button>
          </div>
        </div>
      </div>

      <div id="file-explorer-status"></div>

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
            @navigate-folder="handleNavigateFolder"
            @open-file="handleOpenFile"
            @context-menu="handleContextMenu"
          />
        </template>
        <template v-else>
          <CirrusListView
            :files="files"
            :current-path="currentPath"
            :show-device-badges="showDeviceBadges"
            @navigate-folder="handleNavigateFolder"
            @open-file="handleOpenFile"
            @context-menu="handleContextMenu"
          />
        </template>
      </div>
    </div>

    <!-- File Viewer Modal -->
    <CirrusFileViewer
      v-model="fileViewerOpen"
      :file-path="selectedFilePath"
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
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { CirrusFileNode, FileType } from '@/types/cirrus'
import { getFiles, determineFileType, getFileName } from '@/services/cirrusService'
import CirrusBreadcrumb from './CirrusBreadcrumb.vue'
import CirrusListView from './CirrusListView.vue'
import CirrusGridView from './CirrusGridView.vue'
import CirrusFileViewer from './CirrusFileViewer.vue'
import CirrusContextMenu from './CirrusContextMenu.vue'
import CirrusDropZone from './CirrusDropZone.vue'

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
const selectedFilePath = ref('')
const selectedFileType = ref<FileType>('generic')

// Context menu state
const contextMenuOpen = ref(false)
const contextMenuFile = ref<CirrusFileNode | null>(null)
const contextMenuX = ref(0)
const contextMenuY = ref(0)

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
  const relativePath = currentPath.value ? `${currentPath.value}/${fileName}` : fileName
  selectedFilePath.value = relativePath
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
  const fullPath = currentPath.value ? `${currentPath.value}/${folderName}` : folderName
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
  const relativePath = currentPath.value ? `${currentPath.value}/${fileName}` : fileName
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
  // TODO: Implement rename dialog
  const fileName = getFileName(file)
  console.log('Rename file:', fileName)
  alert(`Rename functionality coming soon for: ${fileName}`)
}

const handleFileDetails = (file: CirrusFileNode) => {
  // TODO: Implement file details dialog
  const fileName = getFileName(file)
  console.log('File details:', fileName)
  alert(`File details for: ${fileName}\nPath: ${file.fullPath}\nDevice: ${file.deviceName}`)
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
  box-shadow: var(--shadow-sm);
  padding: var(--spacing-lg);
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;

  @media (prefers-color-scheme: dark) {
    background-color: var(--color-gray-900);
  }
}

.file-explorer-header {
  display: flex;
  align-items: center;
  margin-bottom: var(--spacing-lg);
}

#file-explorer-selectable {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

#file-explorer-view-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.file-explorer-title {
  font-size: var(--font-size-2xl);
  font-weight: 700;
  margin-right: var(--spacing-lg);
  white-space: nowrap;
  color: var(--color-gray-900);
}
@media (prefers-color-scheme: dark) {
  .file-explorer-title,
  .file-explorer-header,
  .file-explorer-controls,
  .file-explorer-loading,
  .file-explorer-empty,
  .device-badge-toggle,
  .file-explorer-space-info {
    color: white;
  }
  .file-explorer-error {
    color: #f87171;
  }
}

.file-explorer-space-info {
  color: var(--color-gray-500);
}

.file-explorer-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-md);
}

.file-explorer-loading {
  padding: var(--spacing-md) 0;
  color: var(--color-gray-500);
  font-style: italic;
}

.file-explorer-error {
  padding: var(--spacing-md) 0;
  color: #dc2626;

  @media (prefers-color-scheme: dark) {
    color: #f87171;
  }
}

.file-explorer-empty {
  padding: var(--spacing-md) 0;
  color: var(--color-gray-500);
}

.device-badge-toggle {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  font-size: var(--font-size-sm);
  color: var(--color-gray-600);
  cursor: pointer;

  input {
    cursor: pointer;
  }

  @media (prefers-color-scheme: dark) {
    color: var(--color-gray-400);
  }
}

.view-switcher {
  display: flex;
  gap: var(--spacing-xs);
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
  padding: var(--spacing-xs) var(--spacing-sm);
  border-radius: var(--border-radius);
  font-size: var(--font-size-sm);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  border: 1px solid transparent;

  &--icon {
    padding: var(--spacing-xs);
  }

  &--primary {
    background-color: var(--color-primary-600);
    color: white;
    border-color: var(--color-primary-600);

    &:hover {
      background-color: var(--color-primary-400);
    }
  }

  &--secondary {
    background-color: transparent;
    color: var(--color-gray-700);
    border-color: var(--color-gray-300);

    &:hover {
      background-color: var(--color-gray-100);
    }

    @media (prefers-color-scheme: dark) {
      color: var(--color-gray-300);
      border-color: var(--color-gray-600);

      &:hover {
        background-color: var(--color-gray-800);
      }
    }
  }
}
</style>
