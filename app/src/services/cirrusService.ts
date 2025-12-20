// Cirrus service - makes API calls to the backend
import type { CirrusFileNode, FileType } from '@/types/cirrus'

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

/**
 * Extract filename from a full path or use name field
 */
export function getFileName(node: CirrusFileNode): string {
  // Prefer the name field from the API if available
  if (node.name) {
    return node.name
  }
  // Fallback to extracting from fullPath
  const parts = node.fullPath.split('/')
  return parts[parts.length - 1] || ''
}

/**
 * Determine if a node represents a directory
 */
export function isDirectory(node: CirrusFileNode): boolean {
  // Use the isDir field from the API
  return node.isDir
}

/**
 * Get the file size in bytes
 */
export function getFileSize(node: CirrusFileNode): number {
  return node.size || 0
}

/**
 * Determine file type from a CirrusFileNode
 */
export function determineFileType(node: CirrusFileNode): FileType {
  const fileName = getFileName(node)

  if (isDirectory(node)) {
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

/**
 * Fetch files for a given path from the backend API
 */
export async function getFiles(path: string): Promise<CirrusFileNode[]> {
  // Normalize path - remove leading/trailing slashes
  const normalizedPath = path.replace(/^\/+|\/+$/g, '')

  // Build the API URL
  const apiUrl = normalizedPath ? `/api/v1/json/cirrus/${normalizedPath}` : '/api/v1/json/cirrus'

  try {
    const response = await fetch(apiUrl)

    if (!response.ok) {
      throw new Error(`Failed to fetch files: ${response.statusText}`)
    }

    const data: CirrusFileNode[] = await response.json()
    return data
  } catch (error) {
    console.error('Error fetching cirrus files:', error)
    throw error
  }
}

/**
 * Get available space in bytes
 * TODO: This should come from a backend API endpoint
 */
export function getAvailableSpace(): number {
  // Return 100 GB as stub value until we have an API endpoint for this
  return 100 * 1024 * 1024 * 1024
}
