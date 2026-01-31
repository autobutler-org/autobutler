<template>
  <LibraryLayout>
    <template #sidebar>
      <LibrarySidebar :sections="sidebarSections" />
    </template>
    <template #title>
      <h2 class="library-title">Google Migration Service</h2>
    </template>
    <template #subtitle>
      <div class="library-subtitle">
        Configure and start your Google data import
      </div>
    </template>
    <template #main>
      <div class="migration-content">
        <!-- Auth Status -->
        <div class="auth-status-card">
          <div class="auth-status-header">
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
              <h3>Connected as {{ userEmail }}</h3>
              <p>Google account authenticated successfully</p>
            </div>
          </div>
          <button class="btn btn--secondary" @click="disconnectAndReturn">
            Disconnect
          </button>
        </div>

        <!-- Service Selection -->
        <div class="service-selection-card">
          <h3>Select Data to Import</h3>
          <p>Choose which Google services you want to import from:</p>

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
                  <path
                    d="M3 15v4c0 1.1.9 2 2 2h14a2 2 0 0 0 2-2v-4M17 8l-5-5-5 5M12 3v12"
                  />
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
        </div>

        <!-- Device Selection -->
        <div class="device-selection-card">
          <h3>Select Storage Device</h3>
          <p>Where should your Google data be stored?</p>

          <div v-if="loadingDevices" class="loading-devices">
            <div class="spinner"></div>
            <span>Loading devices...</span>
          </div>
          <div v-else-if="devices.length === 0" class="no-devices">
            <p>
              No storage devices available. Please connect a storage device.
            </p>
          </div>
          <div v-else class="device-list">
            <button
              v-for="device in devices"
              :key="device.devicePath"
              :class="[
                'device-option',
                { selected: selectedDevice === device },
              ]"
              @click="selectedDevice = device"
            >
              <div class="device-icon">
                <svg
                  v-if="device.isInternal"
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
                  <line x1="8" y1="21" x2="16" y2="21"></line>
                  <line x1="12" y1="17" x2="12" y2="21"></line>
                </svg>
                <svg
                  v-else
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    d="M20 16v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2M12 4v12M5 11l7-7 7 7"
                  ></path>
                </svg>
              </div>
              <div class="device-info">
                <strong>{{ device.name }}</strong>
                <span class="device-capacity"
                  >{{ formatBytes(device.availableBytes) }} available of
                  {{ formatBytes(device.totalBytes) }}</span
                >
                <span
                  v-if="!device.isInternal && device.usbInfo"
                  class="device-model"
                  >{{ device.usbInfo.manufacturer }}</span
                >
              </div>
              <div class="device-check" v-if="selectedDevice === device">✓</div>
            </button>
          </div>
        </div>

        <!-- Import Actions -->
        <div class="import-actions">
          <button
            class="btn btn--primary btn--large"
            @click="startImport"
            :disabled="!canStartImport || importing"
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
            {{ importing ? 'Starting Import...' : 'Start Import' }}
          </button>
          <RouterLink to="/data-migration" class="btn btn--secondary"
            >Cancel</RouterLink
          >
        </div>

        <!-- Progress Bar -->
        <div v-if="downloadProgress" class="progress-container">
          <div class="progress-header">
            <h3>{{ downloadProgress.status }}</h3>
            <span class="progress-stats">
              {{ downloadProgress.filesProcessed }} /
              {{ downloadProgress.totalFiles }} files
              <span v-if="downloadProgress.totalBytes > 0">
                ({{ formatBytes(downloadProgress.downloadedBytes) }} /
                {{ formatBytes(downloadProgress.totalBytes) }})
              </span>
            </span>
          </div>
          <div class="progress-bar">
            <div
              class="progress-bar-fill"
              :style="{ width: downloadProgress.progress * 100 + '%' }"
            ></div>
          </div>
          <div class="progress-percentage">
            {{ Math.round(downloadProgress.progress * 100) }}%
          </div>
        </div>

        <div v-if="importStatus" :class="['status-message', importStatus.type]">
          {{ importStatus.message }}
        </div>
      </div>
    </template>
  </LibraryLayout>
</template>

<script setup lang="ts">
import LibraryLayout from '@/components/common/LibraryLayout.vue';
import LibrarySidebar from '@/components/common/LibrarySidebar.vue';
import DevicesService from '@/services/devicesService';
import type { Device } from '@/types/device';
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';

const router = useRouter();

const sidebarSections = [
  {
    title: 'Data',
    items: [{ label: 'Data Migration', href: '/data-migration' }],
  },
];

// Auth state
const userEmail = ref('');
const accessToken = ref('');

// Service selection
const selectedServices = ref({
  photos: false,
  drive: false,
  contacts: false,
  calendar: false,
});

// Device selection
const devices = ref<Device[]>([]);
const selectedDevice = ref<Device | null>(null);
const loadingDevices = ref(false);

// Import state
const importing = ref(false);
const importStatus = ref<{
  type: 'success' | 'error' | 'info';
  message: string;
} | null>(null);

const downloadProgress = ref<{
  status: string;
  filesProcessed: number;
  totalFiles: number;
  downloadedBytes: number;
  totalBytes: number;
  progress: number;
} | null>(null);

const hasSelectedServices = computed(() => {
  return Object.values(selectedServices.value).some((selected) => selected);
});

const canStartImport = computed(() => {
  return hasSelectedServices.value && selectedDevice.value !== null;
});

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
};

const checkAuthentication = () => {
  // Check URL hash for OAuth callback params
  if (window.location.hash) {
    const hashParams = new URLSearchParams(window.location.hash.substring(1));
    const email = hashParams.get('email');
    const token = hashParams.get('token');
    const name = hashParams.get('name');

    if (email && token) {
      console.log('Found OAuth params in URL hash, storing...');
      localStorage.setItem('google_auth_email', email);
      localStorage.setItem('google_auth_token', token);
      localStorage.setItem('google_auth_name', name || '');
      localStorage.setItem('google_auth_timestamp', Date.now().toString());
      // Clear hash and reload
      window.location.hash = '';
      window.location.reload();
      return false;
    }
  }

  // Check localStorage for auth data
  const storedEmail = localStorage.getItem('google_auth_email');
  const storedToken = localStorage.getItem('google_auth_token');
  const timestamp = localStorage.getItem('google_auth_timestamp');

  console.log('Checking authentication:', {
    hasEmail: !!storedEmail,
    hasToken: !!storedToken,
    email: storedEmail,
    timestamp: timestamp ? new Date(parseInt(timestamp)).toISOString() : 'none',
  });

  if (!storedEmail || !storedToken) {
    console.log('No authentication found, redirecting to data migration');
    setTimeout(() => router.push('/data-migration'), 2000);
    return false;
  }

  userEmail.value = storedEmail;
  accessToken.value = storedToken;
  console.log('✅ Authenticated as:', userEmail.value);
  return true;
};

const fetchDevices = async () => {
  loadingDevices.value = true;
  try {
    const response = await DevicesService.getDeviceStatuses();
    devices.value = response.devices || [];
    console.log('Loaded devices:', devices.value.length);
    // Auto-select first device if only one available
    if (devices.value.length === 1 && devices.value[0]) {
      selectedDevice.value = devices.value[0];
    }
  } catch (error) {
    console.error('Failed to fetch devices:', error);
    importStatus.value = {
      type: 'error',
      message: 'Failed to load storage devices. Please try again.',
    };
  } finally {
    loadingDevices.value = false;
  }
};

const startImport = async () => {
  if (!canStartImport.value) return;

  importing.value = true;
  importStatus.value = {
    type: 'info',
    message:
      'Starting download from Google... This may take a while for large accounts.',
  };

  try {
    const deviceSerial = selectedDevice.value!.usbInfo?.serial || '';

    const response = await fetch('/api/v1/migration/google/start', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        email: userEmail.value,
        services: selectedServices.value,
        deviceSerial: deviceSerial,
      }),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.message || 'Import failed');
    }

    const data = await response.json();
    const jobId = data.jobId;

    importStatus.value = {
      type: 'success',
      message: `Import started! Downloading your Google data...`,
    };

    // Start polling for status
    pollImportStatus(jobId);
  } catch (error) {
    importStatus.value = {
      type: 'error',
      message: `Failed to start import: ${error instanceof Error ? error.message : 'Unknown error'}`,
    };
    console.error('Import error:', error);
  } finally {
    importing.value = false;
  }
};

const pollImportStatus = async (jobId: string) => {
  // Store jobId in localStorage so we can resume monitoring after page reload
  localStorage.setItem('active_import_job', jobId);

  const pollInterval = setInterval(async () => {
    try {
      const response = await fetch(`/api/v1/migration/google/status/${jobId}`);
      if (!response.ok) {
        clearInterval(pollInterval);
        localStorage.removeItem('active_import_job');
        return;
      }

      const job = await response.json();

      // Update progress bar
      if (job.status === 'DOWNLOADING' || job.status === 'INITIATED') {
        downloadProgress.value = {
          status:
            job.status === 'DOWNLOADING'
              ? 'Downloading from Google...'
              : 'Initializing...',
          filesProcessed: job.filesProcessed || 0,
          totalFiles: job.totalFiles || 0,
          downloadedBytes: job.downloadedBytes || 0,
          totalBytes: job.totalBytes || 0,
          progress: job.progress || 0,
        };
      }

      // Check if completed
      if (job.status === 'COMPLETED') {
        clearInterval(pollInterval);
        localStorage.removeItem('active_import_job');

        downloadProgress.value = null;
        importStatus.value = {
          type: 'success',
          message:
            'Download complete! Your Google data ZIP file is ready in the file explorer.',
        };

        // Reset selections
        selectedServices.value = {
          photos: false,
          drive: false,
          contacts: false,
          calendar: false,
        };
      } else if (job.status === 'FAILED') {
        clearInterval(pollInterval);
        localStorage.removeItem('active_import_job');

        downloadProgress.value = null;
        importStatus.value = {
          type: 'error',
          message: `Import failed: ${job.errorMsg || 'Unknown error'}`,
        };
      }
    } catch (error) {
      console.error('Status poll error:', error);
    }
  }, 2000); // Poll every 2 seconds
};

const disconnectAndReturn = async () => {
  try {
    await fetch('/api/v1/auth/google/disconnect', {
      method: 'POST',
    });
  } catch (error) {
    console.error('Disconnect error:', error);
  }

  // Clear localStorage
  localStorage.removeItem('google_auth_email');
  localStorage.removeItem('google_auth_token');
  localStorage.removeItem('google_auth_name');
  localStorage.removeItem('active_import_job');

  // Redirect to data migration
  router.push('/data-migration');
};

onMounted(() => {
  if (!checkAuthentication()) {
    return;
  }
  fetchDevices();

  // Check for active import job
  const activeJobId = localStorage.getItem('active_import_job');
  if (activeJobId) {
    importing.value = true;
    pollImportStatus(activeJobId);
  }
});
</script>

<style lang="scss" scoped>
.migration-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-2xl);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xl);
  max-width: 900px;
  margin: 0 auto;
}

.auth-status-card {
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.3);
  border-radius: 12px;
  padding: var(--spacing-xl);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--spacing-lg);
}

.auth-status-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  flex: 1;

  .user-avatar {
    width: 48px;
    height: 48px;
    background: rgba(34, 197, 94, 0.2);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;

    svg {
      width: 24px;
      height: 24px;
      color: #22c55e;
    }
  }

  h3 {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: #22c55e;
    margin: 0 0 4px 0;
  }

  p {
    font-size: var(--font-size-sm);
    color: rgba(34, 197, 94, 0.8);
    margin: 0;
  }
}

.service-selection-card,
.device-selection-card {
  background: var(--color-gray-800);
  border-radius: 12px;
  padding: var(--spacing-xl);

  @media (prefers-color-scheme: light) {
    background: white;
    border: 1px solid var(--color-gray-200);
  }

  h3 {
    font-size: var(--font-size-xl);
    font-weight: 600;
    color: white;
    margin: 0 0 var(--spacing-xs) 0;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }

  > p {
    color: var(--color-gray-400);
    margin: 0 0 var(--spacing-lg) 0;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }
  }
}

.service-checkboxes {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.service-checkbox {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-md);
  background: var(--color-gray-750);
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-50);
  }

  &:hover {
    background: var(--color-gray-700);

    @media (prefers-color-scheme: light) {
      background: var(--color-gray-100);
    }
  }

  input[type='checkbox'] {
    width: 20px;
    height: 20px;
    cursor: pointer;
  }

  .checkbox-content {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    flex: 1;

    svg {
      width: 24px;
      height: 24px;
      color: var(--color-primary-500);
    }

    > div {
      display: flex;
      flex-direction: column;
      gap: 4px;

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
}

.loading-devices {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-md);
  padding: var(--spacing-2xl);
  color: var(--color-gray-400);

  .spinner {
    width: 24px;
    height: 24px;
    border: 3px solid var(--color-gray-700);
    border-top-color: var(--color-primary-500);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }
}

.no-devices {
  text-align: center;
  padding: var(--spacing-2xl);
  color: var(--color-gray-400);

  @media (prefers-color-scheme: light) {
    color: var(--color-gray-600);
  }
}

.device-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.device-option {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-lg);
  background: var(--color-gray-750);
  border: 2px solid transparent;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
  width: 100%;

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-50);
  }

  &:hover {
    background: var(--color-gray-700);
    border-color: var(--color-primary-500);

    @media (prefers-color-scheme: light) {
      background: var(--color-gray-100);
    }
  }

  &.selected {
    background: rgba(99, 102, 241, 0.1);
    border-color: var(--color-primary-500);
  }
}

.device-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-gray-700);
  border-radius: 8px;
  flex-shrink: 0;

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-200);
  }

  svg {
    width: 24px;
    height: 24px;
    color: var(--color-gray-300);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }
  }
}

.device-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;

  strong {
    color: white;
    font-size: var(--font-size-base);
    font-weight: 600;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }

  .device-capacity {
    color: var(--color-gray-400);
    font-size: var(--font-size-sm);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }
  }

  .device-model {
    color: var(--color-gray-500);
    font-size: var(--font-size-xs);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-500);
    }
  }
}

.device-check {
  width: 28px;
  height: 28px;
  background: var(--color-primary-500);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  flex-shrink: 0;
}

.import-actions {
  display: flex;
  gap: var(--spacing-md);
  flex-wrap: wrap;
}

.progress-container {
  padding: var(--spacing-xl);
  background: var(--color-gray-750);
  border-radius: 12px;
  border: 2px solid var(--color-primary-500);

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-50);
  }
}

.progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-md);

  h3 {
    color: white;
    font-size: var(--font-size-lg);
    font-weight: 600;
    margin: 0;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }

  .progress-stats {
    color: var(--color-gray-400);
    font-size: var(--font-size-sm);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }
  }
}

.progress-bar {
  width: 100%;
  height: 12px;
  background: var(--color-gray-700);
  border-radius: 6px;
  overflow: hidden;
  margin-bottom: var(--spacing-sm);

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-200);
  }
}

.progress-bar-fill {
  height: 100%;
  background: linear-gradient(
    90deg,
    var(--color-primary-500),
    var(--color-primary-600)
  );
  transition: width 0.3s ease;
  border-radius: 6px;
}

.progress-percentage {
  text-align: right;
  color: var(--color-primary-500);
  font-weight: 600;
  font-size: var(--font-size-base);
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
  padding: var(--spacing-md) var(--spacing-xl);
  border-radius: 8px;
  font-weight: 600;
  font-size: var(--font-size-base);
  cursor: pointer;
  transition: all 0.2s;
  border: none;

  &--large {
    padding: var(--spacing-lg) var(--spacing-2xl);
    font-size: var(--font-size-lg);
  }

  &--primary {
    background: var(--color-primary-600);
    color: white;

    &:hover:not(:disabled) {
      background: var(--color-primary-700);
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  &--secondary {
    background: var(--color-gray-700);
    color: white;

    @media (prefers-color-scheme: light) {
      background: var(--color-gray-200);
      color: var(--color-gray-900);
    }

    &:hover {
      background: var(--color-gray-600);

      @media (prefers-color-scheme: light) {
        background: var(--color-gray-300);
      }
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

@keyframes slideIn {
  from {
    transform: translateX(400px);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

@keyframes slideOut {
  from {
    transform: translateX(0);
    opacity: 1;
  }
  to {
    transform: translateX(400px);
    opacity: 0;
  }
}
</style>
