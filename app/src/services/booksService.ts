// Service for fetching books from the backend
export interface BookApiResponse {
  relPath: string
  fileName: string
  size: number
  mtime: number
  type: string
}

export async function fetchBooks(): Promise<BookApiResponse[]> {
  const res = await fetch('/api/v1/books')
  if (!res.ok) throw new Error('Failed to fetch books')
  return res.json()
}
