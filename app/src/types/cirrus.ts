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
  | 'generic';

// CirrusFileNode matches the JSON response from the backend API
export interface CirrusFileNode {
  name: string;
  size: number;
  isDir: boolean;
  deviceName: string;
  devicePath: string;
  fullPath: string;
  deviceSerial: string;
}

export interface CirrusState {
  files: CirrusFileNode[];
  currentPath: string;
  availableBytes: number;
  view: 'list' | 'grid' | 'column';
  loading: boolean;
  error: string | null;
}
