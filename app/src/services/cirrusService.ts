import type { CirrusFileNode, FileType } from '@/types/cirrus';

import HttpService from './httpService';

export default class CirrusService {
  static bytesToGB(bytes: number): number {
    return bytes / (1024 * 1024 * 1024);
  }

  /**
   * Delete a file in Cirrus
   */
  static async deleteFile(
    rootDir: string,
    fileName: string,
    deviceSerial?: string,
  ): Promise<void> {
    const params = new URLSearchParams();
    params.append('rootDir', rootDir);
    params.append('filePaths', fileName);
    if (deviceSerial) {
      params.append('serial', deviceSerial);
    }
    const url = `/api/v1/cirrus?${params}`;
    const response = await HttpService.delete(url);
    if (!response.ok) throw new Error('Failed to delete file');
  }

  static determineFileType(node: CirrusFileNode): FileType {
    const fileName = CirrusService.getFileName(node);
    if (CirrusService.isDirectory(node)) {
      return 'folder';
    }
    const ext = fileName.toLowerCase().split('.').pop() || '';
    switch (ext) {
      case 'pdf':
        return 'pdf';
      case 'pptx':
      case 'ppt':
        return 'slideshow';
      case 'png':
      case 'jpg':
      case 'jpeg':
      case 'gif':
      case 'svg':
      case 'heic':
      case 'heif':
      case 'webp':
      case 'bmp':
      case 'tiff':
      case 'tif':
      case 'avif':
        return 'image';
      case 'mp4':
      case 'm4v':
      case 'webm':
      case 'ogg':
      case 'avi':
      case 'mov':
        return 'video';
      case 'epub':
        return 'epub';
      case 'docx':
        return 'docx';
      case 'zip':
      case 'rar':
      case 'tar':
      case 'gz':
      case '7z':
        return 'archive';
      default:
        return 'generic';
    }
  }

  // Utility methods can be static or exported as needed
  static formatBytes(bytes: number): string {
    if (bytes < 1024) {
      return `${bytes} B`;
    } else if (bytes < 1024 * 1024) {
      return `${(bytes / 1024).toFixed(1)} KB`;
    } else if (bytes < 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    } else if (bytes < 1024 * 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    } else {
      return `${(bytes / (1024 * 1024 * 1024 * 1024)).toFixed(1)} TB`;
    }
  }

  /**
   * Get available space in bytes (mocked)
   */
  static getAvailableSpace(): number {
    return 100 * 1024 * 1024 * 1024;
  }

  /**
   * Construct a download URL for a Cirrus file. Clients may use this
   * directly as an `src` for iframes or anchors.
   */
  static getDownloadUrl(filePath: string, serial?: string): string {
    const params = new URLSearchParams({ filePath });
    if (serial) {
      params.append('serial', serial);
    }
    return `/api/v1/cirrus/download?${params}`;
  }

  /**
   * Fetch files for a given path from the backend API
   */
  static async getFiles(path: string): Promise<CirrusFileNode[]> {
    const normalizedPath = CirrusService.normalizePath(path);
    return await HttpService.getAsJson<CirrusFileNode[]>(
      '/api/v1/cirrus',
      normalizedPath
        ? new URLSearchParams({ rootDir: normalizedPath })
        : undefined,
    );
  }

  static getFileName(node: CirrusFileNode): string {
    if (node.name) {
      return node.name;
    }
    const parts = node.fullPath.split('/');
    return parts[parts.length - 1] || '';
  }

  static getFileSize(node: CirrusFileNode): number {
    return node.size || 0;
  }

  static isDirectory(node: CirrusFileNode): boolean {
    return node.isDir;
  }

  /**
   * Move or rename a file or directory in Cirrus
   */
  static async moveFile(
    oldPath: string,
    newPath: string,
    oldDeviceSerial?: string,
    newDeviceSerial?: string,
  ): Promise<void> {
    await HttpService.put('/api/v1/cirrus', {
      oldFilePath: oldPath,
      newFilePath: newPath,
      oldDeviceSerial: oldDeviceSerial,
      newDeviceSerial: newDeviceSerial,
    });
  }

  static normalizePath(path: string): string {
    return path.replace(/^\/+|\/+$/g, '').trim();
  }

  /**
   * Upload files to Cirrus
   */
  static async uploadFiles(
    uploadPath: string,
    files: FileList | File[],
    serial?: string,
  ): Promise<Response> {
    const formData = new FormData();
    for (const file of Array.from(files)) {
      formData.append('files', file);
    }
    const url =
      (uploadPath.startsWith('/') ? uploadPath : '/' + uploadPath) +
      (serial ? `?serial=${encodeURIComponent(serial)}` : '');
    const response = await HttpService.postForm(url, formData);
    if (!response.ok) throw new Error('Upload failed');
    return response;
  }
}
