#!/usr/bin/env node

const fs = require('fs')
const { execSync } = require('child_process')
const path = require('path')

/**
 * Generate build information file with commit details
 * This script should be run during the build process
 */
const generateBuildInfo = () => {
  const buildInfo = {
    buildDate: new Date().toISOString(),
    version: process.env.npm_package_version || '1.0.0'
  }

  try {
    // Get git commit information
    const commitHash = execSync('git rev-parse HEAD', { encoding: 'utf8' }).trim()
    const commitDate = execSync('git log -1 --format=%cd --date=iso-strict', { encoding: 'utf8' }).trim()
    const commitMessage = execSync('git log -1 --format=%s', { encoding: 'utf8' }).trim()

    buildInfo.commitHash = commitHash
    buildInfo.commitDate = commitDate
    buildInfo.commitMessage = commitMessage
  } catch (error) {
    console.warn('Could not retrieve git information:', error.message)
  }

  // Write to public directory so it's accessible at runtime
  const publicDir = path.join(__dirname, '..', 'public')
  if (!fs.existsSync(publicDir)) {
    fs.mkdirSync(publicDir, { recursive: true })
  }

  const buildInfoPath = path.join(publicDir, 'build-info.json')
  fs.writeFileSync(buildInfoPath, JSON.stringify(buildInfo, null, 2))

  console.log('Build info generated:', buildInfo)
  console.log('Saved to:', buildInfoPath)
}

// Run if called directly
if (require.main === module) {
  generateBuildInfo()
}

module.exports = { generateBuildInfo } 