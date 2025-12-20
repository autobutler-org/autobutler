// Types for Cirrus file explorer

export type FileType =
  | 'folder'
  | 'pdf'
  | 'slideshow'
  | 'image'
  | 'video'
  | 'epub'
  | 'docx'
  | 'archive'
  | 'generic'

// CirrusFileNode matches the JSON response from the backend API
// The fileInfo field is an empty object when serialized (fs.FileInfo doesn't serialize to JSON)
export interface CirrusFileNode {
  fileInfo: Record<string, unknown> // Empty object from backend
  deviceName: string
  devicePath: string
  fullPath: string
}

export interface CirrusState {
  files: CirrusFileNode[]
  currentPath: string
  availableBytes: number
  view: 'list' | 'grid' | 'column'
  loading: boolean
  error: string | null
}
