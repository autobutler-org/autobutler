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

export interface DeviceFileInfo {
  name: string
  size: number
  isDir: boolean
  modTime: string
  deviceName: string
  devicePath: string
  fullPath: string
}

export interface CirrusState {
  files: DeviceFileInfo[]
  currentPath: string
  availableBytes: number
  view: 'list' | 'grid' | 'column'
  loading: boolean
  error: string | null
}
