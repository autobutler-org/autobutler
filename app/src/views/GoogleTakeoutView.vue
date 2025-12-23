<template>
  <LibraryLayout>
    <template #sidebar>
      <LibrarySidebar :sections="sidebarSections" />
      <div class="settings-sidebar-thanks">
        <h2>Thanks</h2>
        <a href="/thanks" class="settings-thanks-link">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            width="20"
            height="20"
          >
            <path
              d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"
            ></path>
          </svg>
          <span>View Thanks</span>
        </a>
      </div>
    </template>
    <template #title>
      <h2 class="library-title">Google Takeout Import</h2>
    </template>
    <template #subtitle>
      <div class="library-subtitle">Upload and import your Google Takeout data</div>
    </template>
    <template #main>
      <div class="migration-content">
        <div class="takeout-header">
          <div class="takeout-icon">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
              <path
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
              />
              <path
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              />
            </svg>
          </div>
          <div>
            <h3>Automated Google Import</h3>
            <p>
              Connect your Google account and AutoButler will automatically request and import your
              data.
            </p>
          </div>
        </div>

        <!-- Authentication Section -->
        <div v-if="!isAuthenticated" class="auth-section">
          <div class="auth-card">
            <h3>How It Works</h3>
            <div class="process-steps">
              <div class="process-step">
                <div class="process-number">1</div>
                <div class="process-text">
                  <strong>Authenticate</strong>
                  <span>Sign in with your Google account securely</span>
                </div>
              </div>
              <div class="process-step">
                <div class="process-number">2</div>
                <div class="process-text">
                  <strong>Request Export</strong>
                  <span>AutoButler automatically requests your data from Google Takeout</span>
                </div>
              </div>
              <div class="process-step">
                <div class="process-number">3</div>
                <div class="process-text">
                  <strong>Auto-Download</strong>
                  <span>When ready, your data is automatically downloaded and imported</span>
                </div>
              </div>
            </div>

            <p class="auth-description">
              AutoButler will request access to your Google Photos and Drive. You can review and
              revoke these permissions at any time from your Google Account settings.
            </p>
            <div class="permissions-list">
              <h4>Requested Permissions:</h4>
              <ul>
                <li>
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    width="16"
                    height="16"
                  >
                    <polyline points="20 6 9 17 4 12"></polyline>
                  </svg>
                  Read access to Google Photos
                </li>
                <li>
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    width="16"
                    height="16"
                  >
                    <polyline points="20 6 9 17 4 12"></polyline>
                  </svg>
                  Read access to Google Drive
                </li>
                <li>
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    width="16"
                    height="16"
                  >
                    <polyline points="20 6 9 17 4 12"></polyline>
                  </svg>
                  Basic profile information
                </li>
              </ul>
            </div>
            <button class="btn btn--google" @click="initiateGoogleAuth" :disabled="authenticating">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="currentColor"
                width="20"
                height="20"
              >
                <path
                  d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                />
                <path
                  d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                />
                <path
                  d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                />
                <path
                  d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                />
              </svg>
              {{ authenticating ? 'Connecting...' : 'Sign in with Google' }}
            </button>
            <p class="privacy-note">
              We never store your Google password. Authentication is handled securely through
              Google's OAuth 2.0 protocol.
            </p>
          </div>
        </div>

        <!-- Data Selection Section -->
        <div v-else class="selection-section">
          <div class="user-info">
            <div class="user-avatar">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"></path>
                <circle cx="12" cy="7" r="4"></circle>
              </svg>
            </div>
            <div>
              <h4>Connected as {{ userEmail }}</h4>
              <button class="btn-link" @click="disconnectGoogle">Disconnect</button>
            </div>
          </div>

          <div class="data-selection-card">
            <h3>Select Data to Import</h3>
            <p>AutoButler will automatically request, download, and import your selected data:</p>

            <div class="service-checkboxes">
              <label class="service-checkbox">
                <input type="checkbox" v-model="selectedServices.photos" />
                <div class="checkbox-content">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                    <circle cx="8.5" cy="8.5" r="1.5" />
                    <path d="m21 15-5-5L5 21" />
                  </svg>
                  <div>
                    <strong>Google Photos</strong>
                    <span>Import all your photos and videos</span>
                  </div>
                </div>
              </label>

              <label class="service-checkbox">
                <input type="checkbox" v-model="selectedServices.drive" />
                <div class="checkbox-content">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path d="M3 15v4c0 1.1.9 2 2 2h14a2 2 0 0 0 2-2v-4M17 8l-5-5-5 5M12 3v12" />
                  </svg>
                  <div>
                    <strong>Google Drive</strong>
                    <span>Import your files and folders</span>
                  </div>
                </div>
              </label>

              <label class="service-checkbox">
                <input type="checkbox" v-model="selectedServices.contacts" />
                <div class="checkbox-content">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
                    <circle cx="9" cy="7" r="4" />
                    <path d="M23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75" />
                  </svg>
                  <div>
                    <strong>Contacts</strong>
                    <span>Import your contact list</span>
                  </div>
                </div>
              </label>

              <label class="service-checkbox">
                <input type="checkbox" v-model="selectedServices.calendar" />
                <div class="checkbox-content">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                    <line x1="16" y1="2" x2="16" y2="6" />
                    <line x1="8" y1="2" x2="8" y2="6" />
                    <line x1="3" y1="10" x2="21" y2="10" />
                  </svg>
                  <div>
                    <strong>Calendar</strong>
                    <span>Import your events and reminders</span>
                  </div>
                </div>
              </label>
            </div>

            <div class="import-actions">
              <button
                class="btn btn--primary"
                @click="startImport"
                :disabled="!hasSelectedServices || importing"
              >
                <svg
                  v-if="importing"
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  width="20"
                  height="20"
                  class="spinner"
                >
                  <line x1="12" y1="2" x2="12" y2="6"></line>
                  <line x1="12" y1="18" x2="12" y2="22"></line>
                  <line x1="4.93" y1="4.93" x2="7.76" y2="7.76"></line>
                  <line x1="16.24" y1="16.24" x2="19.07" y2="19.07"></line>
                  <line x1="2" y1="12" x2="6" y2="12"></line>
                  <line x1="18" y1="12" x2="22" y2="12"></line>
                  <line x1="4.93" y1="19.07" x2="7.76" y2="16.24"></line>
                  <line x1="16.24" y1="7.76" x2="19.07" y2="4.93"></line>
                </svg>
                {{ importing ? 'Importing...' : 'Start Import' }}
              </button>
              <RouterLink to="/data-migration" class="btn btn--secondary">
                Back to Migration Options
              </RouterLink>
            </div>
          </div>

          <div v-if="importStatus" :class="['status-message', importStatus.type]">
            {{ importStatus.message }}
          </div>
        </div>
      </div>
    </template>
  </LibraryLayout>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { RouterLink } from 'vue-router'
import LibraryLayout from '@/components/common/LibraryLayout.vue'
import LibrarySidebar from '@/components/common/LibrarySidebar.vue'

const sidebarSections = [
  {
    title: 'Sections',
    items: [
      { label: 'General', href: '/settings' },
      { label: 'Users & Access' },
      { label: 'Storage' },
      { label: 'Networking' },
      { label: 'Security' },
      { label: 'OpenTelemetry' },
      { label: 'Advanced' },
    ],
  },
  {
    title: 'Data',
    items: [{ label: 'Data Migration', href: '/data-migration' }],
  },
]

const isAuthenticated = ref(false)
const authenticating = ref(false)
const userEmail = ref('')
const importing = ref(false)
const importStatus = ref<{ type: 'success' | 'error' | 'info'; message: string } | null>(null)

interface SelectedServices {
  photos: boolean
  drive: boolean
  contacts: boolean
  calendar: boolean
}

const selectedServices = ref<SelectedServices>({
  photos: false,
  drive: false,
  contacts: false,
  calendar: false,
})

const hasSelectedServices = computed(() => {
  return Object.values(selectedServices.value).some((selected) => selected)
})

const initiateGoogleAuth = async () => {
  authenticating.value = true
  importStatus.value = null

  try {
    // Request OAuth authorization URL from backend
    const response = await fetch('/api/v1/auth/google/authorize', {
      method: 'GET',
    })

    if (!response.ok) {
      throw new Error('Failed to initiate authentication')
    }

    const data = await response.json()

    // Open OAuth popup
    window.open(data.authUrl, 'Google Authentication', 'width=600,height=700,left=100,top=100')

    // Listen for OAuth callback
    window.addEventListener('message', handleAuthCallback)
  } catch (error) {
    importStatus.value = {
      type: 'error',
      message: 'Failed to connect to Google. Please try again.',
    }
    console.error('Auth error:', error)
  } finally {
    authenticating.value = false
  }
}

const handleAuthCallback = async (event: MessageEvent) => {
  if (event.origin !== window.location.origin) return

  if (event.data.type === 'google-auth-success') {
    isAuthenticated.value = true
    userEmail.value = event.data.email
    window.removeEventListener('message', handleAuthCallback)
    importStatus.value = {
      type: 'success',
      message: 'Successfully connected to Google!',
    }
  } else if (event.data.type === 'google-auth-error') {
    importStatus.value = {
      type: 'error',
      message: 'Authentication failed. Please try again.',
    }
    window.removeEventListener('message', handleAuthCallback)
  }
}

const disconnectGoogle = async () => {
  try {
    await fetch('/api/v1/auth/google/disconnect', {
      method: 'POST',
    })
    isAuthenticated.value = false
    userEmail.value = ''
    selectedServices.value = {
      photos: false,
      drive: false,
      contacts: false,
      calendar: false,
    }
    importStatus.value = {
      type: 'info',
      message: 'Disconnected from Google.',
    }
  } catch (error) {
    console.error('Disconnect error:', error)
  }
}

const startImport = async () => {
  if (!hasSelectedServices.value) return

  importing.value = true
  importStatus.value = {
    type: 'info',
    message:
      'Requesting your data from Google Takeout... This process happens in the background and may take several hours for large accounts.',
  }

  try {
    // Backend will:
    // 1. Use Google Takeout API to request export with selected services
    // 2. Poll for export completion
    // 3. Download zip files when ready
    // 4. Automatically upload to /api/v1/cirrus endpoint
    // 5. Worker processes the files asynchronously
    const response = await fetch('/api/v1/migration/google/start', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        services: selectedServices.value,
      }),
    })

    if (!response.ok) {
      throw new Error('Import failed')
    }

    const data = await response.json()

    importStatus.value = {
      type: 'success',
      message: `Export requested successfully! Export ID: ${data.exportId}. Google is preparing your data. You'll be notified when the download and import begins. This typically takes 2-24 hours depending on data size.`,
    }

    // Reset selections
    selectedServices.value = {
      photos: false,
      drive: false,
      contacts: false,
      calendar: false,
    }
  } catch (error) {
    importStatus.value = {
      type: 'error',
      message:
        'Failed to request export. Please try again or contact support if the issue persists.',
    }
    console.error('Import error:', error)
  } finally {
    importing.value = false
  }
}
</script>

<style lang="scss" scoped>
.migration-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-2xl);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3xl);
}

.takeout-header {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-lg);
  padding: var(--spacing-2xl);
  background: var(--color-gray-800);
  border-radius: 12px;

  @media (prefers-color-scheme: light) {
    background: white;
    border: 2px solid var(--color-gray-200);
  }

  h3 {
    margin: 0 0 var(--spacing-sm) 0;
    font-size: var(--font-size-xl);
    font-weight: 600;
    color: white;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }

  p {
    margin: 0;
    color: var(--color-gray-400);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }
  }
}

.takeout-icon {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 193, 7, 0.15);
  border-radius: 8px;

  svg {
    width: 28px;
    height: 28px;
    color: #ffc107;
  }
}

.instructions-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.instruction-step {
  display: flex;
  gap: var(--spacing-lg);
  padding: var(--spacing-xl);
  background: var(--color-gray-800);
  border-radius: 12px;

  @media (prefers-color-scheme: light) {
    background: white;
    border: 2px solid var(--color-gray-200);
  }
}

.step-number {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary-600);
  color: white;
  border-radius: 50%;
  font-weight: 600;
  font-size: var(--font-size-sm);
}

.step-content {
  flex: 1;

  h4 {
    margin: 0 0 var(--spacing-xs) 0;
    font-size: var(--font-size-base);
    font-weight: 600;
    color: white;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }

  p {
    margin: 0;
    color: var(--color-gray-400);
    font-size: var(--font-size-sm);
    line-height: 1.6;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }
  }
}

.external-link {
  color: var(--color-primary-500);
  text-decoration: none;
  font-weight: 500;

  &:hover {
    text-decoration: underline;
  }
}

.auth-section {
  padding: var(--spacing-2xl);
  background: var(--color-gray-800);
  border-radius: 12px;

  @media (prefers-color-scheme: light) {
    background: white;
    border: 2px solid var(--color-gray-200);
  }
}

.auth-card {
  max-width: 700px;
  margin: 0 auto;

  h3 {
    margin: 0 0 var(--spacing-xl) 0;
    color: white;
    font-size: var(--font-size-xl);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }
}

.process-steps {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
  margin-bottom: var(--spacing-2xl);
}

.process-step {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg);
  background: rgba(37, 99, 235, 0.05);
  border-left: 3px solid var(--color-primary-600);
  border-radius: 8px;

  @media (prefers-color-scheme: light) {
    background: var(--color-blue-50);
  }
}

.process-number {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary-600);
  color: white;
  border-radius: 50%;
  font-weight: 700;
  font-size: var(--font-size-lg);
}

.process-text {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);

  strong {
    color: white;
    font-size: var(--font-size-base);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }

  span {
    color: var(--color-gray-400);
    font-size: var(--font-size-sm);
    line-height: 1.5;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }
  }
}

.auth-description {
  margin: 0 0 var(--spacing-xl) 0;
  color: var(--color-gray-400);
  line-height: 1.6;
  font-size: var(--font-size-sm);

  @media (prefers-color-scheme: light) {
    color: var(--color-gray-600);
  }
}

.permissions-list {
  margin: var(--spacing-2xl) 0;
  padding: var(--spacing-lg);
  background: rgba(255, 255, 255, 0.03);
  border-radius: 8px;

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-50);
  }

  h4 {
    margin: 0 0 var(--spacing-md) 0;
    color: white;
    font-size: var(--font-size-sm);
    font-weight: 600;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }

  ul {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: var(--spacing-sm);
  }

  li {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    color: var(--color-gray-300);
    font-size: var(--font-size-sm);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-700);
    }

    svg {
      color: var(--color-green-500);
      flex-shrink: 0;
    }
  }
}

.btn--google {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-md);
  padding: var(--spacing-md) var(--spacing-xl);
  background: white;
  color: #1f2937;
  border: 1px solid var(--color-gray-300);
  border-radius: 8px;
  font-weight: 600;
  font-size: var(--font-size-base);
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover:not(:disabled) {
    background: var(--color-gray-50);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  svg {
    width: 20px;
    height: 20px;
  }
}

.privacy-note {
  margin-top: var(--spacing-lg);
  font-size: var(--font-size-xs);
  color: var(--color-gray-500);
  text-align: center;
  line-height: 1.5;
}

.selection-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2xl);
}

.user-info {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-lg);
  background: var(--color-gray-800);
  border-radius: 8px;

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-50);
    border: 1px solid var(--color-gray-200);
  }

  h4 {
    margin: 0 0 var(--spacing-xs) 0;
    color: white;
    font-size: var(--font-size-base);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }
}

.user-avatar {
  width: 48px;
  height: 48px;
  background: var(--color-primary-600);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;

  svg {
    width: 24px;
    height: 24px;
    color: white;
  }
}

.btn-link {
  background: none;
  border: none;
  color: var(--color-primary-500);
  font-size: var(--font-size-sm);
  cursor: pointer;
  padding: 0;
  text-decoration: underline;

  &:hover {
    color: var(--color-primary-400);
  }
}

.data-selection-card {
  padding: var(--spacing-2xl);
  background: var(--color-gray-800);
  border-radius: 12px;

  @media (prefers-color-scheme: light) {
    background: white;
    border: 2px solid var(--color-gray-200);
  }

  h3 {
    margin: 0 0 var(--spacing-sm) 0;
    color: white;
    font-size: var(--font-size-xl);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }

  > p {
    margin: 0 0 var(--spacing-xl) 0;
    color: var(--color-gray-400);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }
  }
}

.service-checkboxes {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
  margin-bottom: var(--spacing-xl);
}

.service-checkbox {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-md);
  padding: var(--spacing-lg);
  background: rgba(255, 255, 255, 0.03);
  border: 2px solid transparent;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-50);
    border-color: var(--color-gray-200);
  }

  &:hover {
    background: rgba(37, 99, 235, 0.05);
    border-color: var(--color-primary-500);
  }

  input[type='checkbox'] {
    margin-top: 4px;
    cursor: pointer;
    width: 18px;
    height: 18px;
    accent-color: var(--color-primary-600);
  }
}

.checkbox-content {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-md);
  flex: 1;

  svg {
    width: 24px;
    height: 24px;
    color: var(--color-primary-500);
    flex-shrink: 0;
    margin-top: 2px;
  }

  > div {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);

    strong {
      color: white;
      font-size: var(--font-size-base);

      @media (prefers-color-scheme: light) {
        color: var(--color-gray-900);
      }
    }

    span {
      color: var(--color-gray-400);
      font-size: var(--font-size-sm);

      @media (prefers-color-scheme: light) {
        color: var(--color-gray-600);
      }
    }
  }
}

.import-actions {
  display: flex;
  gap: var(--spacing-md);
  flex-wrap: wrap;
}

.status-message {
  padding: var(--spacing-lg);
  border-radius: 8px;
  font-weight: 500;

  &.success {
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
    border: 1px solid rgba(34, 197, 94, 0.3);
  }

  &.error {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border: 1px solid rgba(239, 68, 68, 0.3);
  }

  &.info {
    background: rgba(59, 130, 246, 0.1);
    color: #3b82f6;
    border: 1px solid rgba(59, 130, 246, 0.3);
  }
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-md) var(--spacing-lg);
  border-radius: 8px;
  font-size: var(--font-size-base);
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  border: none;
  text-decoration: none;

  &--primary {
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-700) 100%);
    color: white;
    box-shadow: 0 4px 14px rgba(37, 99, 235, 0.3);

    &:hover:not(:disabled) {
      background: linear-gradient(
        135deg,
        var(--color-primary-700) 0%,
        var(--color-primary-800) 100%
      );
      box-shadow: 0 6px 20px rgba(37, 99, 235, 0.4);
    }

    &:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    .spinner {
      animation: spin 1s linear infinite;
    }
  }

  &--secondary {
    background: transparent;
    color: var(--color-primary-500);
    border: 2px solid var(--color-primary-500);

    &:hover {
      background: rgba(37, 99, 235, 0.1);
    }
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.settings-sidebar-thanks {
  margin-top: var(--spacing-2xl);
  padding: var(--spacing-md);
  border-top: 1px solid var(--color-gray-800);

  h2 {
    font-size: var(--font-size-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--color-gray-400);
    margin-bottom: var(--spacing-md);
  }
}

.settings-thanks-link {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  color: var(--color-primary-600);
  text-decoration: none;
  font-weight: 600;
  margin-top: var(--spacing-xs);

  &:hover {
    text-decoration: underline;
  }
}
</style>
