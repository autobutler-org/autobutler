import type { CirrusFileNode, FileType } from '@/types/cirrus';

import { joinPaths } from '@/util/filepath';
import HttpService from './httpService';

export default class CirrusService {
  static bytesToGB(bytes: number): number {
    return bytes / (1024 * 1024 * 1024);
  }

  static async createFolder(
    folderPath: string,
    folderName: string,
  ): Promise<void> {
    const formData = new FormData();
    formData.append('folderName', folderName);
    const url = folderPath
      ? `/api/v1/cirrus/folder/${folderPath}`
      : '/api/v1/cirrus/folder/';
    const response = await HttpService.postForm(url, formData);
    if (!response.ok) throw new Error('Failed to create folder');
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
  static async getFiles(
    path: string,
    serials?: string[],
  ): Promise<CirrusFileNode[]> {
    const normalizedPath = CirrusService.normalizePath(path);
    const params = normalizedPath
      ? new URLSearchParams({ rootDir: normalizedPath })
      : new URLSearchParams();
    if (serials && serials.length > 0) {
      for (const s of serials) {
        // append each serial as repeated query param
        params.append('serial', s);
      }
    }
    const useParams =
      params && params.toString().length > 0 ? params : undefined;
    return await HttpService.getAsJson<CirrusFileNode[]>(
      '/api/v1/cirrus',
      useParams,
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
    return CirrusService.uploadFilesFromFormData(uploadPath, formData, serial);
  }

  /**
   * Upload files to Cirrus with formData
   */
  static async uploadFilesFromFormData(
    uploadPath: string,
    formData: FormData,
    serial?: string,
  ): Promise<Response> {
    const url = `${joinPaths('/api/v1/cirrus/upload', uploadPath)}${serial ? `?${new URLSearchParams({ serial })}` : ''}`;
    const response = await HttpService.postForm(url, formData);
    if (!response.ok) throw new Error('Upload failed');
    return response;
  }
}
