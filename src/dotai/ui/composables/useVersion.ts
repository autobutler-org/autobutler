import { computed, ref, readonly } from 'vue'
import packageJson from '../package.json'

// Types for build info
interface BuildInfo {
  buildDate: string
  commitHash?: string
  commitDate?: string
}

// Functional utility to format date as YYYYMMDD
const formatDateAsVersion = (date: Date): string => 
  date.toISOString().slice(0, 10).replace(/-/g, '')

// Functional utility to parse version and replace minor with date
const createVersionWithDate = (version: string, buildDate: Date): string => {
  const [major] = version.split('.')
  const dateMinor = formatDateAsVersion(buildDate)
  return `${major}.${dateMinor}.0`
}

// Functional utility to get build info from various sources
const getBuildDate = async (): Promise<Date> => {
  // Try to get build info from build artifact first
  try {
    const buildInfo = await $fetch<BuildInfo>('/build-info.json')
    return new Date(buildInfo.commitDate || buildInfo.buildDate)
  } catch {
    // If no build info file, try git on server side
    return getLastCommitDate()
  }
}

// Functional utility to get last commit date (fallback to current date)
const getLastCommitDate = async (): Promise<Date> => {
  try {
    // Try to get last commit date from git
    if (process.client) {
      // On client side, we'll use the build date
      return new Date()
    }
    
    // On server side, try to get git info
    const { execSync } = await import('child_process')
    const gitDate = execSync('git log -1 --format=%cd --date=iso-strict', { 
      encoding: 'utf8',
      cwd: process.cwd() 
    }).trim()
    
    return new Date(gitDate)
  } catch {
    // Fallback to current date if git is not available or command fails
    return new Date()
  }
}

export const useVersion = () => {
  const buildDate = ref<Date>(new Date())
  
  const version = computed(() => 
    createVersionWithDate(packageJson.version, buildDate.value)
  )
  
  const displayVersion = computed(() => `v${version.value}`)
  
  const initializeVersion = async () => {
    buildDate.value = await getBuildDate()
  }
  
  // Initialize on first use
  initializeVersion()
  
  return {
    version: readonly(version),
    displayVersion: readonly(displayVersion),
    buildDate: readonly(buildDate),
    refresh: initializeVersion
  }
} 