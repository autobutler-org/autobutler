import { ref, unref, type Ref } from 'vue';

import CirrusService from '@/services/cirrusService';
import { useCirrusDeviceStore } from '@/stores/cirrusDeviceStore';
import type { CirrusFileNode } from '@/types/cirrus';
import { joinPathsNormalized } from '@/util/filepath';

type MaybeRef<T> = T | Ref<T>;

interface UseCirrusFileDropZoneOptions {
  currentPath: MaybeRef<string>;
  onFilesUploaded: (files: CirrusFileNode[]) => void;
}

export const useCirrusFileDropZone = ({
  currentPath,
  onFilesUploaded,
}: UseCirrusFileDropZoneOptions) => {
  const isDragOver = ref(false);
  const isUploading = ref(false);
  const uploadProgress = ref('');
  const fileInputRef = ref<HTMLInputElement | null>(null);

  const resolveCurrentPath = () =>
    CirrusService.normalizePath(unref(currentPath) || '');

  const uploadFiles = async (files: FileList, targetPath?: string) => {
    if (!files.length) return;

    isUploading.value = true;
    uploadProgress.value = `Uploading ${files.length} file${files.length > 1 ? 's' : ''}...`;

    try {
      const targetUploadPath = targetPath
        ? CirrusService.normalizePath(targetPath)
        : resolveCurrentPath();

      // Get selected device serial and name from Pinia store
      const cirrusDeviceStore = useCirrusDeviceStore();
      const serial = cirrusDeviceStore.selectedDeviceSerial || '';
      const selectedDevice =
        cirrusDeviceStore.devices.find(
          (d) => (d.usbInfo?.serial || '') === serial,
        ) || cirrusDeviceStore.devices.find((d) => !d.usbInfo?.serial);
      const deviceName = selectedDevice ? selectedDevice.name : '';

      await CirrusService.uploadFiles(targetUploadPath, files, serial);

      const currentPathValue = targetUploadPath;

      const uploadedNodes: CirrusFileNode[] = Array.from(files).map((file) => ({
        name: file.name,
        size: file.size,
        isDir: false,
        deviceName,
        devicePath: '',
        fullPath: currentPathValue
          ? joinPathsNormalized(currentPathValue, file.name)
          : file.name,
        deviceSerial: serial,
      }));

      uploadProgress.value = 'Upload complete!';
      onFilesUploaded(uploadedNodes);

      setTimeout(() => {
        uploadProgress.value = '';
      }, 2000);
    } catch (error) {
      uploadProgress.value = `Upload failed: ${
        error instanceof Error ? error.message : 'Unknown error'
      }`;

      setTimeout(() => {
        uploadProgress.value = '';
      }, 3000);
    } finally {
      isUploading.value = false;
    }
  };

  const handleDragEnter = (event: DragEvent) => {
    event.preventDefault();
    isDragOver.value = isDraggingFiles(event);
  };

  const handleDragOver = (event: DragEvent) => {
    event.preventDefault();
    isDragOver.value = isDraggingFiles(event);
  };

  const handleDragLeave = (event: DragEvent) => {
    event.preventDefault();

    const currentTarget = event.currentTarget as Node | null;
    const relatedTarget = event.relatedTarget as Node | null;

    // Ignore leaves that move within the current drop zone
    if (
      currentTarget &&
      relatedTarget &&
      currentTarget.contains(relatedTarget)
    ) {
      return;
    }

    isDragOver.value = false;
  };

  const handleDrop = async (event: DragEvent, targetPath?: string) => {
    event.preventDefault();
    isDragOver.value = false;

    // If this is a file move, let the directory drop handler handle it
    if (event.dataTransfer?.getData('application/x-cirrus-file-path')) {
      return;
    }

    const files = event.dataTransfer?.files;
    if (files && files.length) {
      await uploadFiles(files, targetPath);
    }
  };

  const handleClick = () => {
    fileInputRef.value?.click();
  };

  const handleFileInputChange = (event: Event) => {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      uploadFiles(input.files);
      input.value = '';
    }
  };

  const isDraggingFiles = (event: DragEvent) => {
    if (event.dataTransfer?.types) {
      for (const type of event.dataTransfer.types) {
        if (type === 'Files') {
          return true;
        }
      }
    }
    return false;
  };

  return {
    fileInputRef,
    handleClick,
    handleDragEnter,
    handleDragLeave,
    handleDragOver,
    handleDrop,
    handleFileInputChange,
    isDragOver,
    isDraggingFiles,
    isUploading,
    uploadProgress,
  };
};

export type UseCirrusFileDropZoneReturn = ReturnType<
  typeof useCirrusFileDropZone
>;
