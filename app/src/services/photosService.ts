// Service for fetching photos from the backend
export interface PhotoApiResponse {
  relPath: string
  fileName: string
  size: number
  mtime: string
}

export async function fetchPhotos(): Promise<PhotoApiResponse[]> {
  const res = await fetch('/api/v1/photos')
  if (!res.ok) throw new Error('Failed to fetch photos')
  return res.json()
}
