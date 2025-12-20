// Stubbed Cirrus service - returns static data instead of fetch calls
import type { DeviceFileInfo } from '@/types/cirrus'

// Helper to format bytes to human readable string
export function formatBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`
  } else if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`
  } else if (bytes < 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  } else if (bytes < 1024 * 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
  } else {
    return `${(bytes / (1024 * 1024 * 1024 * 1024)).toFixed(1)} TB`
  }
}

export function bytesToGB(bytes: number): number {
  return bytes / (1024 * 1024 * 1024)
}

// Stub data for files in root directory
const stubRootFiles: DeviceFileInfo[] = [
  {
    name: 'Documents',
    size: 1024 * 1024 * 500, // 500 MB
    isDir: true,
    modTime: '2025-01-15T10:30:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Documents',
  },
  {
    name: 'Photos',
    size: 1024 * 1024 * 1024 * 2, // 2 GB
    isDir: true,
    modTime: '2025-01-10T14:22:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Photos',
  },
  {
    name: 'Videos',
    size: 1024 * 1024 * 1024 * 5, // 5 GB
    isDir: true,
    modTime: '2025-01-05T09:15:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Videos',
  },
  {
    name: 'report.pdf',
    size: 1024 * 1024 * 2.5, // 2.5 MB
    isDir: false,
    modTime: '2025-01-20T16:45:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/report.pdf',
  },
  {
    name: 'presentation.pptx',
    size: 1024 * 1024 * 15, // 15 MB
    isDir: false,
    modTime: '2025-01-18T11:30:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/presentation.pptx',
  },
  {
    name: 'vacation.jpg',
    size: 1024 * 1024 * 4, // 4 MB
    isDir: false,
    modTime: '2025-01-12T08:00:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/vacation.jpg',
  },
  {
    name: 'archive.zip',
    size: 1024 * 1024 * 50, // 50 MB
    isDir: false,
    modTime: '2025-01-08T13:20:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/archive.zip',
  },
  {
    name: 'notes.docx',
    size: 1024 * 256, // 256 KB
    isDir: false,
    modTime: '2025-01-19T17:00:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/notes.docx',
  },
]

// Stub data for files in Documents subdirectory
const stubDocumentsFiles: DeviceFileInfo[] = [
  {
    name: 'Work',
    size: 1024 * 1024 * 200, // 200 MB
    isDir: true,
    modTime: '2025-01-14T09:00:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Documents/Work',
  },
  {
    name: 'Personal',
    size: 1024 * 1024 * 100, // 100 MB
    isDir: true,
    modTime: '2025-01-13T11:30:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Documents/Personal',
  },
  {
    name: 'contract.pdf',
    size: 1024 * 512, // 512 KB
    isDir: false,
    modTime: '2025-01-16T10:15:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Documents/contract.pdf',
  },
  {
    name: 'resume.docx',
    size: 1024 * 128, // 128 KB
    isDir: false,
    modTime: '2025-01-17T14:45:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Documents/resume.docx',
  },
]

// Stub data for files in Photos subdirectory
const stubPhotosFiles: DeviceFileInfo[] = [
  {
    name: 'Summer 2024',
    size: 1024 * 1024 * 800, // 800 MB
    isDir: true,
    modTime: '2024-08-20T16:00:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Photos/Summer 2024',
  },
  {
    name: 'beach.jpg',
    size: 1024 * 1024 * 5, // 5 MB
    isDir: false,
    modTime: '2024-07-15T12:30:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Photos/beach.jpg',
  },
  {
    name: 'sunset.png',
    size: 1024 * 1024 * 8, // 8 MB
    isDir: false,
    modTime: '2024-07-16T19:45:00Z',
    deviceName: 'Main Storage',
    devicePath: '/',
    fullPath: '/Photos/sunset.png',
  },
]

// Map of path to stub files
const stubFilesMap: Record<string, DeviceFileInfo[]> = {
  '': stubRootFiles,
  '/': stubRootFiles,
  Documents: stubDocumentsFiles,
  '/Documents': stubDocumentsFiles,
  Photos: stubPhotosFiles,
  '/Photos': stubPhotosFiles,
}

/**
 * Get files for a given path (stubbed implementation)
 * In the real implementation, this would make a fetch call to the backend API
 */
export function getFiles(path: string): DeviceFileInfo[] {
  // Normalize path - remove leading/trailing slashes for lookup
  const normalizedPath = path.replace(/^\/+|\/+$/g, '')
  return stubFilesMap[normalizedPath] || stubFilesMap[path] || []
}

/**
 * Get available space in bytes (stubbed implementation)
 * In the real implementation, this would come from the backend
 */
export function getAvailableSpace(): number {
  // Return 100 GB as stub value
  return 100 * 1024 * 1024 * 1024
}

/**
 * Determine file type from filename
 */
export function determineFileType(
  fileName: string,
  isDir: boolean,
): 'folder' | 'pdf' | 'slideshow' | 'image' | 'video' | 'epub' | 'docx' | 'archive' | 'generic' {
  if (isDir) {
    return 'folder'
  }

  const ext = fileName.toLowerCase().split('.').pop() || ''

  switch (ext) {
    case 'pdf':
      return 'pdf'
    case 'pptx':
    case 'ppt':
      return 'slideshow'
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
      return 'image'
    case 'mp4':
    case 'm4v':
    case 'webm':
    case 'ogg':
    case 'avi':
    case 'mov':
      return 'video'
    case 'epub':
      return 'epub'
    case 'docx':
      return 'docx'
    case 'zip':
    case 'rar':
    case 'tar':
    case 'gz':
    case '7z':
      return 'archive'
    default:
      return 'generic'
  }
}
