/**
 * Version service for fetching app version information
 */

export interface Release {
  tagName: string
  name: string
  htmlUrl: string
  publishedAt: string
  isCurrentVersion: boolean
}

export interface VersionResponse {
  semver: string
  gitCommit: string
  goVersion: string
  buildDate: string
}

/**
 * Fetch the current version from the API
 */
export const getCurrentVersion = async (): Promise<string> => {
  try {
    const response = await fetch('/api/v1/version')
    if (!response.ok) {
      return 'vX.Y.Z'
    }
    const data: VersionResponse = await response.json()
    return data.semver || 'vX.Y.Z'
  } catch {
    return 'vX.Y.Z'
  }
}

/**
 * Fetch available releases for the version dropdown
 */
export const getAvailableReleases = async (): Promise<Release[]> => {
  try {
    const response = await fetch('/api/v1/versions')
    if (!response.ok) {
      return []
    }
    const data: Release[] = await response.json()
    return data || []
  } catch {
    return []
  }
}
