<template>
  <div id="file-explorer" class="file-explorer">
    <div class="file-explorer-header">
      <div class="file-explorer-header-row">
        <div class="file-explorer-upload-row header-left">
          <template v-if="devices.length > 1">
            <select
              v-model="selectedDeviceSerial"
              class="device-select"
              :disabled="isUploading || devices.length === 0"
              aria-label="Select upload location"
            >
              <option
                v-for="device in devices"
                :key="device.name"
                :value="device.usbInfo?.serial || ''"
              >
                {{ device.name }}
              </option>
            </select>
          </template>
          <span v-if="uploadProgress" class="upload-progress">{{
            uploadProgress
          }}</span>
          <button
            class="action-btn toolbar-rect upload-rect"
            type="button"
            :disabled="isUploading || devices.length === 0"
            @click="handleUploadClick"
            title="Upload files"
            aria-label="Upload files"
          >
            <UploadIcon />
          </button>
          <button
            class="action-btn"
            type="button"
            :disabled="selectedFiles.length === 0 || isUploading"
            @click="handleDownloadSelected"
            title="Download selected files"
            aria-label="Download selected files"
          >
            <DownloadIcon />
          </button>
          <button
            class="action-btn"
            type="button"
            :disabled="selectedFiles.length === 0 || isUploading"
            @click="handleDeleteSelected"
            title="Delete selected files"
            aria-label="Delete selected files"
          >
            <DeleteIcon />
          </button>
          <input
            ref="fileInputRef"
            type="file"
            multiple
            style="display: none"
            @change="handleFileInputChange"
            aria-label="Select files to upload"
          />
        </div>

        <h2 class="file-explorer-title centered-title">Cirrus</h2>

        <div class="view-switcher header-right">
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

    <div id="file-explorer-selectable">
      <div class="file-explorer-controls">
        <div>
          <CirrusBreadcrumbs
            :current-path="currentPath"
            @navigate="navigateToPath"
            @folder-created="handleFolderCreated"
            @breadcrumb-drop="handleBreadcrumbDrop"
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
            <span>Show devices</span>
          </label>
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
        <template v-else>
          <div
            class="file-table-container"
            style="
              display: flex;
              flex-direction: column;
              flex: 1;
              min-height: 0;
            "
          >
            <component
              :is="fileViewComponent"
              :files="files"
              :current-path="currentPath"
              :show-device-badges="showDeviceBadges"
              :selected-files="selectedFiles"
              @navigate-folder="handleNavigateFolder"
              @open-file="handleOpenFile"
              @select="handleSelectFile"
              @context-menu="handleContextMenu"
              @files-uploaded="handleFilesUploaded"
              @deselect-all="handleDeselectAll"
              @file-move="onFileMove"
            />
          </div>
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
    <ModalDialog
      v-if="moveDialogOpen"
      @close="moveDialogOpen = false"
      :transparent="true"
      :hide-close-button="true"
    >
      <div class="custom-modal-close-wrapper">
        <button
          class="custom-modal-close"
          @click="moveDialogOpen = false"
          aria-label="Close"
        >
          <CloseIcon />
        </button>
      </div>
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
        <div class="move-dialog-field" v-if="devices.length > 1">
          <label for="move-device-select" class="move-dialog-label"
            >Target device</label
          >
          <select
            id="move-device-select"
            v-model="moveDialogTargetDeviceSerial"
            :disabled="moveDialogLoading"
            class="device-select"
            aria-label="Select target device"
          >
            <option
              v-for="device in devices"
              :key="device.usbInfo?.serial || 'internal'"
              :value="device.usbInfo?.serial || ''"
            >
              {{ device.name }}
            </option>
          </select>
        </div>
        <div v-if="moveDialogError" class="move-dialog-error">
          {{ moveDialogError }}
        </div>
        <div class="move-dialog-actions">
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
    <!-- File Details Modal Dialog -->
    <ModalDialog
      v-if="detailsDialogOpen"
      @close="closeDetailsDialog"
      :transparent="true"
      :hide-close-button="true"
    >
      <div class="custom-modal-close-wrapper">
        <button
          class="custom-modal-close"
          @click="closeDetailsDialog"
          aria-label="Close"
        >
          <CloseIcon />
        </button>
      </div>
      <div class="details-dialog-form">
        <div class="details-dialog-header">
          <span class="details-dialog-title">File Details</span>
        </div>
        <div v-if="detailsDialogFile">
          <table class="details-table">
            <tbody>
              <tr>
                <th>Name</th>
                <td>{{ detailsDialogFile.name }}</td>
              </tr>
              <tr>
                <th>Path</th>
                <td class="details-path">{{ detailsDialogFile.fullPath }}</td>
              </tr>
              <tr>
                <th>Device</th>
                <td>{{ detailsDialogFile.deviceName }}</td>
              </tr>
              <tr v-if="detailsDialogFile.size !== undefined">
                <th>Size</th>
                <td>{{ detailsDialogFile.size }} bytes</td>
              </tr>
              <tr>
                <th>Type</th>
                <td>{{ detailsDialogFile.isDir ? 'Folder' : 'File' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </ModalDialog>
    <!-- Delete Confirmation Modal Dialog -->
    <ModalDialog
      v-if="deleteDialogOpen"
      @close="closeDeleteDialog"
      :transparent="true"
      :hide-close-button="true"
    >
      <div class="custom-modal-close-wrapper">
        <button
          class="custom-modal-close"
          @click="closeDeleteDialog"
          aria-label="Close"
        >
          <CloseIcon />
        </button>
      </div>
      <form @submit.prevent="confirmDelete" class="move-dialog-form">
        <h3 class="move-dialog-title">Delete File</h3>
        <div class="move-dialog-field">
          <template v-if="deleteDialogFiles.length > 1">
            <span
              >Are you sure you want to delete
              {{ deleteDialogFiles.length }} files?</span
            >
            <ul
              style="
                max-height: 200px;
                overflow: auto;
                margin: 0.5rem 0;
                padding-left: 1rem;
              "
            >
              <li v-for="f in deleteDialogFiles" :key="f.fullPath">
                {{ f.name }}
              </li>
            </ul>
          </template>
          <template v-else>
            <span
              >Are you sure you want to delete "{{
                (deleteDialogFiles.length > 0
                  ? deleteDialogFiles[0]
                  : deleteDialogFile
                )?.name
              }}"?</span
            >
          </template>
        </div>
        <div v-if="deleteDialogError" class="move-dialog-error">
          {{ deleteDialogError }}
        </div>
        <div class="move-dialog-actions">
          <button
            type="submit"
            class="btn btn--primary"
            :disabled="deleteDialogLoading"
          >
            <span v-if="deleteDialogLoading">Deleting...</span>
            <span v-else>Delete</span>
          </button>
        </div>
      </form>
    </ModalDialog>
  </div>
</template>

<script lang="ts" setup>
import ModalDialog from '@/components/common/ModalDialog.vue';
import CloseIcon from '@/components/icons/CloseIcon.vue';
import DeleteIcon from '@/components/icons/DeleteIcon.vue';
import DownloadIcon from '@/components/icons/DownloadIcon.vue';
import UploadIcon from '@/components/icons/UploadIcon.vue';
import CirrusService from '@/services/cirrusService';
import type {
  CirrusFileMoveParams,
  CirrusFileNode,
  FileType,
} from '@/types/cirrus';
import {
  getFileNameFromPath,
  joinPathsNormalized,
  normalizePath,
} from '@/util/filepath';
import { computed, onMounted, ref, ref as vueRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import GridViewIcon from '../icons/GridViewIcon.vue';
import ListViewIcon from '../icons/ListViewIcon.vue';
import CirrusBreadcrumbs from './CirrusBreadcrumbs.vue';
import CirrusContextMenu from './CirrusContextMenu.vue';
import CirrusFileViewer from './CirrusFileViewer.vue';
import CirrusGridView from './CirrusGridView.vue';
import CirrusListView from './CirrusListView.vue';

import DevicesService from '@/services/devicesService';
import { useCirrusDeviceStore } from '@/stores/cirrusDeviceStore';
import { storeToRefs } from 'pinia';

const fileInputRef = vueRef<HTMLInputElement | null>(null);
const isUploading = vueRef(false);
const uploadProgress = vueRef('');
const cirrusDeviceStore = useCirrusDeviceStore();
const { devices, selectedDeviceSerial } = storeToRefs(cirrusDeviceStore);

const fileViewComponent = computed(() => {
  switch (view.value) {
    case 'grid':
      return CirrusGridView;
    case 'list':
    default:
      return CirrusListView;
  }
});

const handleBreadcrumbDrop = async (event: DragEvent, targetPath: string) => {
  const dt = event.dataTransfer;
  const multi = dt?.getData('application/x-cirrus-multi');
  let moves: CirrusFileMoveParams[] = [];
  if (multi) {
    try {
      const filesArr = JSON.parse(multi) as {
        path: string;
        deviceSerial: string | undefined;
      }[];
      moves = filesArr.map((f) => {
        const fileName = getFileNameFromPath(f.path || '');
        const newTargetPath = joinPathsNormalized(targetPath, fileName);
        return {
          oldPath: f.path,
          newPath: newTargetPath,
          oldDeviceSerial: f.deviceSerial,
          newDeviceSerial: f.deviceSerial,
        };
      });
    } catch {}
  } else {
    // Single file fallback
    const fullPath = dt?.getData('application/x-cirrus-file-path');
    if (!fullPath) return;
    const fileName = getFileNameFromPath(fullPath || '');
    const newTargetPath = joinPathsNormalized(targetPath, fileName);
    const deviceSerial =
      dt?.getData('application/x-cirrus-device-serial') || undefined;
    moves = [
      {
        oldPath: fullPath,
        newPath: newTargetPath,
        oldDeviceSerial: deviceSerial,
        newDeviceSerial: deviceSerial,
      },
    ];
  }
  await handleFileMove(moves);
};

// Unselect all files when CirrusListView emits deselect-all
const handleDeselectAll = () => {
  selectedFiles.value = [];
  lastSelectedFile.value = null;
};
// Delete all selected files (open confirmation dialog)
const handleDeleteSelected = async () => {
  if (selectedFiles.value.length === 0) return;
  // Open the confirmation dialog for bulk deletion
  openDeleteDialog();
};

// Download all selected files
const handleDownloadSelected = () => {
  if (selectedFiles.value.length === 0) return;
  for (const file of selectedFiles.value) {
    const fileName = CirrusService.getFileName(file);
    const relativePath = currentPath.value
      ? joinPathsNormalized(currentPath.value, fileName)
      : fileName;
    const downloadUrl = `/api/v1/download/cirrus/${relativePath}${
      file.deviceSerial
        ? `?serial=${encodeURIComponent(file.deviceSerial)}`
        : ''
    }`;
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = fileName;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }
};

const handleUploadClick = () => {
  fileInputRef.value?.click();
};

const handleFileInputChange = async (event: Event) => {
  const input = event.target as HTMLInputElement;
  if (!input.files || input.files.length === 0) return;
  isUploading.value = true;
  uploadProgress.value = `Uploading ${input.files.length} file${input.files.length > 1 ? 's' : ''}...`;
  try {
    // Actually upload the files to the backend, include device as query param
    const uploadPath = `/api/v1/cirrus/${currentPath.value || ''}`;
    await CirrusService.uploadFiles(
      uploadPath,
      input.files,
      selectedDeviceSerial.value,
    );
    uploadProgress.value = 'Upload complete!';
    // Refresh the file list after upload
    await fetchFiles();
  } catch {
    uploadProgress.value = 'Upload failed';
  } finally {
    isUploading.value = false;
    setTimeout(() => {
      uploadProgress.value = '';
    }, 1500);
    input.value = '';
  }
};
// Fetch available storage devices
const fetchDevices = async () => {
  try {
    const resp = await DevicesService.getDeviceStatuses();
    cirrusDeviceStore.setDevices(resp.devices || []);
  } catch {
    cirrusDeviceStore.setDevices([]);
  }
};

onMounted(() => {
  fetchFiles();
  fetchDevices();
});

const route = useRoute();
const router = useRouter();

// State
const currentPath = ref('');
const files = ref<CirrusFileNode[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const view = ref<'list' | 'grid'>('list');
const showDeviceBadges = ref(true);

// File viewer state
const fileViewerOpen = ref(false);
const selectedFileSrc = ref('');
const selectedFileType = ref<FileType>('generic');
// Selection state
const selectedFiles = ref<CirrusFileNode[]>([]);
const lastSelectedFile = ref<CirrusFileNode | null>(null);

// Context menu state
const contextMenuOpen = ref(false);
const contextMenuFile = ref<CirrusFileNode | null>(null);
const contextMenuX = ref(0);
const contextMenuY = ref(0);

// Move/Rename dialog state
const moveDialogOpen = ref(false);
const moveDialogLoading = ref(false);
const moveDialogError = ref('');
const moveDialogNewPath = ref('');
const moveDialogOldDeviceSerial = ref('');
const moveDialogTargetDeviceSerial = ref('');
const moveDialogFile = ref<CirrusFileNode | null>(null);
// File details dialog state
const detailsDialogOpen = ref(false);
const detailsDialogFile = ref<CirrusFileNode | null>(null);

const openDetailsDialog = (file: CirrusFileNode) => {
  detailsDialogFile.value = file;
  detailsDialogOpen.value = true;
};
const closeDetailsDialog = () => {
  detailsDialogOpen.value = false;
  detailsDialogFile.value = null;
};

// Delete dialog state
const deleteDialogOpen = ref(false);
const deleteDialogLoading = ref(false);
const deleteDialogError = ref('');
const deleteDialogFile = ref<CirrusFileNode | null>(null);
const deleteDialogFiles = ref<CirrusFileNode[]>([]);

const openDeleteDialog = (file?: CirrusFileNode) => {
  deleteDialogError.value = '';
  if (file) {
    deleteDialogFile.value = file;
    deleteDialogFiles.value = [];
  } else {
    // bulk delete: copy current selection
    deleteDialogFiles.value = selectedFiles.value.slice();
    deleteDialogFile.value = null;
  }
  deleteDialogOpen.value = true;
};
const closeDeleteDialog = () => {
  deleteDialogOpen.value = false;
  deleteDialogFile.value = null;
  deleteDialogFiles.value = [];
  deleteDialogError.value = '';
};

const confirmDelete = async () => {
  // Determine targets: either the files array (bulk) or single file
  const targets =
    deleteDialogFiles.value.length > 0
      ? deleteDialogFiles.value
      : deleteDialogFile.value
        ? [deleteDialogFile.value]
        : [];
  if (targets.length === 0) return;
  deleteDialogLoading.value = true;
  deleteDialogError.value = '';
  try {
    for (const file of targets) {
      const fileName = CirrusService.getFileName(file);
      await CirrusService.deleteFile(
        currentPath.value,
        fileName,
        file.deviceSerial,
      );
      // Remove from in-memory lists
      files.value = files.value.filter((f) => f.fullPath !== file.fullPath);
      selectedFiles.value = selectedFiles.value.filter(
        (f) => f.fullPath !== file.fullPath,
      );
    }
    closeDeleteDialog();
  } catch (e) {
    deleteDialogError.value =
      e instanceof Error ? e.message : 'Failed to delete file(s)';
  } finally {
    deleteDialogLoading.value = false;
  }
};

// Fetch files for the current path
const fetchFiles = async () => {
  loading.value = true;
  error.value = null;
  try {
    files.value = await CirrusService.getFiles(currentPath.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load files';
    files.value = [];
  } finally {
    loading.value = false;
  }
};

// Watch route changes to update current path
watch(
  () => {
    return route.params.pathMatch;
  },
  (newPath) => {
    if (Array.isArray(newPath)) {
      currentPath.value = newPath.join('/');
    } else {
      currentPath.value = newPath || '';
    }
  },
  { immediate: true },
);

// Watch current path changes to fetch files
watch(currentPath, () => {
  fetchFiles();
});

// Fetch files on mount
onMounted(() => {
  fetchFiles();
});

// Methods
// TODO: Move to a common utility file
const constructFileSrc = (relativePath: string) =>
  `/api/v1/download/cirrus/${relativePath}`;

const handleSelectFile = (file: CirrusFileNode, event?: MouseEvent) => {
  // Multi-select logic: ctrl/cmd for toggle, shift for range, else single select
  if (event && (event.ctrlKey || event.metaKey)) {
    // Toggle selection
    const idx = selectedFiles.value.findIndex(
      (f) => f.fullPath === file.fullPath,
    );
    if (idx >= 0) {
      selectedFiles.value.splice(idx, 1);
    } else {
      selectedFiles.value.push(file);
    }
    lastSelectedFile.value = file;
  } else if (event && event.shiftKey && lastSelectedFile.value) {
    // Range select (from lastSelectedFile to clicked)
    const filesList = files.value;
    const startIdx = filesList.findIndex(
      (f) => f.fullPath === lastSelectedFile.value?.fullPath,
    );
    const endIdx = filesList.findIndex((f) => f.fullPath === file.fullPath);
    if (startIdx >= 0 && endIdx >= 0) {
      const [from, to] =
        startIdx < endIdx ? [startIdx, endIdx] : [endIdx, startIdx];
      const range = filesList.slice(from, to + 1);
      // Replace selection with range
      selectedFiles.value = range;
    }
    lastSelectedFile.value = file;
  } else {
    // Single select
    selectedFiles.value = [file];
    lastSelectedFile.value = file;
  }
};
const navigateToPath = (path: string) => {
  currentPath.value = path;
  router.push(`/cirrus${path ? '/' + path : ''}`);
};

const handleNavigateFolder = (path: string) => {
  navigateToPath(path);
};

const handleOpenFile = (file: CirrusFileNode) => {
  // Construct the relative path for the API from currentPath and filename
  const fileName = CirrusService.getFileName(file);
  const relativePath = currentPath.value
    ? joinPathsNormalized(currentPath.value, fileName)
    : fileName;
  selectedFileSrc.value = constructFileSrc(relativePath);
  selectedFileType.value = CirrusService.determineFileType(file);
  fileViewerOpen.value = true;
};

const switchView = (newView: 'list' | 'grid') => {
  view.value = newView;
};

const toggleDeviceBadges = (show: boolean) => {
  showDeviceBadges.value = show;
};

const getParentPath = (fullPath: string) => {
  const segments = normalizePath(fullPath).split('/').filter(Boolean);
  segments.pop();
  return segments.join('/');
};

// Handle files uploaded - add them to the list
const handleFilesUploaded = (uploadedFiles: CirrusFileNode[]) => {
  const currentPathNormalized = normalizePath(currentPath.value);

  for (const newFile of uploadedFiles) {
    const normalizedFullPath = normalizePath(newFile.fullPath);
    const parentPath = getParentPath(newFile.fullPath);

    // Only display the file if it belongs to the directory currently in view
    if (parentPath !== currentPathNormalized) {
      continue;
    }

    const exists = files.value.some((f) => {
      return normalizePath(f.fullPath) === normalizedFullPath;
    });

    if (!exists) {
      files.value.push({
        ...newFile,
        fullPath: normalizedFullPath,
      });
    }
  }
};

// Handle folder created - add it to the list
const handleFolderCreated = (folderName: string) => {
  const fullPath = currentPath.value
    ? joinPathsNormalized(currentPath.value, folderName)
    : folderName;
  const exists = files.value.some((f) => {
    return f.fullPath === fullPath;
  });
  if (!exists) {
    // Add folder at the beginning (folders typically appear first)
    files.value.unshift({
      name: folderName,
      size: 0,
      isDir: true,
      deviceName: '',
      devicePath: '',
      fullPath,
      deviceSerial: '',
    });
  }
};

// Context menu handlers
const handleContextMenu = (event: MouseEvent, file: CirrusFileNode) => {
  contextMenuFile.value = file;
  contextMenuX.value = event.clientX;
  contextMenuY.value = event.clientY;
  contextMenuOpen.value = true;
};

const handleDownload = (file: CirrusFileNode) => {
  const fileName = CirrusService.getFileName(file);
  const relativePath = currentPath.value
    ? joinPathsNormalized(currentPath.value, fileName)
    : fileName;
  const downloadUrl = `/api/v1/download/cirrus/${relativePath}${
    file.deviceSerial ? `?serial=${encodeURIComponent(file.deviceSerial)}` : ''
  }`;

  // Create a temporary link and click it to trigger download
  const link = document.createElement('a');
  link.href = downloadUrl;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
};

// Accepts an array of move params
const handleFileMove = async (moves: CirrusFileMoveParams[]) => {
  try {
    // Batch update files list: remove files that were moved out of the current folder
    const currentFolder = normalizePath(currentPath.value);
    const movedOutOldNames = moves
      .filter((move) => {
        const newFolder = normalizePath(
          move.newPath.split('/').slice(0, -1).join('/'),
        );
        return newFolder !== currentFolder;
      })
      .map((move) => move.oldPath.split('/').pop() || move.oldPath);
    if (movedOutOldNames.length > 0) {
      files.value = files.value.filter(
        (f) => !movedOutOldNames.includes(normalizePath(f.name)),
      );
    }
    // Actually perform the file move
    await Promise.all(
      moves.map(async (move) => {
        await CirrusService.moveFile(
          move.oldPath,
          move.newPath,
          move.oldDeviceSerial,
          move.newDeviceSerial,
        );
      }),
    );
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to move file(s)';
  }
};
// Adapter for child components: always expects an array
const onFileMove = (moves: CirrusFileMoveParams[]) => {
  handleFileMove(moves);
};

const handleRename = (file: CirrusFileNode) => {
  moveDialogFile.value = file;
  moveDialogError.value = '';
  // Show the full path including current directory context
  const currentPathValue = normalizePath(currentPath.value);
  moveDialogNewPath.value = currentPathValue
    ? joinPathsNormalized(currentPathValue, file.name)
    : file.name;
  moveDialogOpen.value = true;
  moveDialogOldDeviceSerial.value = file.deviceSerial;
  // Default target device to current device
  moveDialogTargetDeviceSerial.value = file.deviceSerial;
};

const submitMoveDialog = async () => {
  if (!moveDialogFile.value) return;
  moveDialogLoading.value = true;
  moveDialogError.value = '';
  try {
    // Include the current path context when constructing the old path
    const fileName = moveDialogFile.value.name;
    const oldPath = currentPath.value
      ? joinPathsNormalized(currentPath.value, fileName)
      : fileName;
    const newPathInput = moveDialogNewPath.value.trim();

    // Construct the new full path (in current directory unless it includes a path separator)
    const newPath = newPathInput.includes('/')
      ? newPathInput
      : currentPath.value
        ? joinPathsNormalized(currentPath.value, newPathInput)
        : newPathInput;

    if (
      !newPathInput ||
      (newPathInput === fileName &&
        moveDialogTargetDeviceSerial.value === moveDialogOldDeviceSerial.value)
    ) {
      moveDialogError.value = 'Please enter a new name or device.';
      moveDialogLoading.value = false;
      return;
    }
    await CirrusService.moveFile(
      normalizePath(oldPath),
      normalizePath(newPath),
      moveDialogOldDeviceSerial.value,
      moveDialogTargetDeviceSerial.value,
    );
    // Refresh the file list after move
    await fetchFiles();
    // Close the dialog
    moveDialogOpen.value = false;
    moveDialogFile.value = null;
  } catch (e) {
    moveDialogError.value =
      e instanceof Error ? e.message : 'Failed to move file.';
  } finally {
    moveDialogLoading.value = false;
  }
};

const handleFileDetails = (file: CirrusFileNode) => {
  openDetailsDialog(file);
};

const handleDelete = (file: CirrusFileNode) => {
  openDeleteDialog(file);
};
</script>

<style lang="scss" scoped>
.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: $theme-palette-bg-nav;
  color: $theme-palette-accent;
  font-size: 1.5rem;
  border-radius: $border-radius-lg;
  padding: 0.5rem 1rem;
  min-width: 40px;
  min-height: 40px;
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s,
    border-color 0.15s;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  &:hover:not(:disabled) {
    background: $theme-palette-bg-secondary;
    color: $theme-palette-text-inverse;
  }
  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  svg {
    width: 1.5rem;
    height: 1.5rem;
    display: block;
  }
}

// Empty area for deselecting in list view
.file-explorer-deselect-area {
  background: transparent;
}
.file-explorer {
  max-width: 100%;
  box-shadow: $shadow-sm;
  padding: $spacing-lg;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.file-explorer-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-md;
  color: $theme-palette-text-primary;
}

.file-explorer-error {
  padding: $spacing-md 0;
  color: $theme-palette-danger;
}

.file-explorer-empty {
  padding: $spacing-md 0;
  color: $theme-palette-text-muted;
}

.file-explorer-header {
  display: flex;
  align-items: center;
  margin-bottom: $spacing-lg;
  color: $theme-palette-text-primary;
}

.file-explorer-header-row {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  margin-bottom: $spacing-xs;
  justify-content: space-between;
  flex-wrap: nowrap;
  width: 100%;
}

.header-left {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
}

.centered-title {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  white-space: nowrap;
  margin: 0;
  z-index: 1;
}

.header-right {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.toolbar-rect {
  padding: 0.5rem 1rem;
  min-width: 44px;
  min-height: 40px;
  border-radius: $border-radius-md;
  background: #ffffff;
  color: $theme-palette-text-primary;
  border: 1px solid $theme-palette-border-strong;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.toolbar-rect.active {
  background-color: $theme-palette-accent;
  color: $theme-palette-text-inverse;
  border-color: $theme-palette-accent;
}

.file-explorer-loading {
  padding: $spacing-md 0;
  color: $theme-palette-text-muted;
  font-style: italic;
}

#file-explorer-selectable {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.file-explorer-title {
  font-size: $theme-font-size-2xl;
  font-weight: 700;
  margin-right: $spacing-lg;
  white-space: nowrap;
  color: $theme-palette-text-primary;
}

.file-explorer-upload-row {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
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
  font-size: $theme-font-size-sm;
  color: $theme-palette-text-muted;
  cursor: pointer;

  input {
    cursor: pointer;
  }
}

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

  &--primary {
    background-color: $theme-palette-accent;
    color: $theme-palette-text-inverse;
    border-color: $theme-palette-accent;

    &:hover {
      background-color: $theme-palette-accent-hover;
    }
  }

  &--secondary {
    background-color: transparent;
    color: $theme-palette-text-primary;
    border-color: $theme-palette-border-strong;

    &:hover {
      background-color: $theme-palette-bg-secondary;
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
  color: $theme-palette-error;
  font-size: $theme-font-size-sm;
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
  background: $theme-palette-bg-secondary;
  border-radius: $border-radius-lg;
  box-shadow: $shadow-lg;
  padding: $spacing-xl $spacing-lg $spacing-lg $spacing-lg;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: $spacing-lg;

  button.btn {
    min-width: 110px;
    font-size: $theme-font-size-base;
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
  border: 1.5px solid $theme-palette-border-strong;
  border-radius: $border-radius-md;
  font-size: $theme-font-size-lg;
  width: 100%;
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
  background: $theme-palette-bg-inverse;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.01);

  &:focus {
    outline: none;
    border-color: $theme-palette-accent;
    box-shadow: 0 0 0 2px $theme-palette-accent-hover;
  }

  &::placeholder {
    color: $theme-palette-text-muted;
    opacity: 1;
  }
}

.move-dialog-label {
  font-size: $theme-font-size-base;
  font-weight: 500;
  color: $theme-palette-text-secondary;
  margin-bottom: $spacing-xs;
}

.move-dialog-title {
  font-size: $theme-font-size-xl;
  font-weight: 700;
  margin-bottom: $spacing-md;
  text-align: left;
  color: $theme-palette-text-primary;
}

.view-switcher {
  display: flex;
  gap: $spacing-xs;
  align-items: center;

  .btn {
    min-width: 44px;
    min-height: 40px;
    border-radius: $border-radius-md;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: $spacing-xs;
  }
}

/* File Details Dialog Styles */
.details-dialog-form {
  min-width: 420px;
  max-width: 98vw;
  background: $theme-palette-bg-secondary;
  border-radius: $border-radius-lg;
  box-shadow: $shadow-lg;
  padding: $spacing-xl $spacing-lg $spacing-lg $spacing-lg;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: $spacing-lg;
}

.details-dialog-header {
  margin-bottom: $spacing-md;
}

.details-dialog-title {
  font-size: $theme-font-size-2xl;
  font-weight: 700;
  color: $theme-palette-text-primary;
  letter-spacing: 0.01em;
}

.details-table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0 0.5rem;
  font-size: $theme-font-size-lg;
  color: $theme-palette-text-primary;
}
.details-table th {
  text-align: left;
  font-weight: 600;
  color: $theme-palette-text-secondary;
  padding-right: 1.5rem;
  vertical-align: top;
  min-width: 80px;
}
.details-table td {
  word-break: break-all;
  color: $theme-palette-text-primary;
}
.details-path {
  font-size: $theme-font-size-base;
  color: $theme-palette-text-muted;
}

.custom-modal-close-wrapper {
  position: relative;
}

.custom-modal-close {
  position: absolute;
  top: 1.5rem;
  right: 1.5rem;
  background: $theme-palette-bg-inverse;
  border: none;
  border-radius: 50%;
  color: $theme-palette-text-inverse;
  cursor: pointer;
  z-index: 1100;
  box-shadow: 0 2px 8px rgba($theme-palette-bg-primary, 0.1);
  width: 2.5rem;
  height: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
  padding: 0;
  svg {
    display: block;
    margin: auto;
  }
  &:hover {
    background: $theme-palette-accent-hover;
  }
}

.device-select {
  padding: 0.4rem 0.7rem;
  border-radius: $border-radius-md;
  border: 0.09rem solid $theme-palette-border-strong;
  font-size: $theme-font-size-base;
  background: $theme-palette-bg-inverse;
  min-width: 7.5rem;
  max-width: 13.75rem;
  margin-right: 0.5rem;
}
</style>
