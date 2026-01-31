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
      <h2 class="library-title">Data Migration</h2>
    </template>
    <template #subtitle>
      <div class="library-subtitle">
        Import your data from Google, iCloud, Dropbox, OneDrive, Box, and more
      </div>
    </template>
    <template #main>
      <div class="migration-content">
        <div class="migration-container">
          <!-- Google Takeout Option -->
          <div class="migration-card google-takeout">
            <div class="migration-card-header">
              <div class="migration-card-icon google">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="currentColor"
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
              </div>
              <h3>Google</h3>
            </div>
            <p class="migration-card-description">
              Automatically download and import your Google Photos, Drive files,
              contacts, and calendar events directly from your Google Account.
            </p>
            <div class="migration-card-features">
              <div class="feature-item">
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
                <span>Photos & Videos</span>
              </div>
              <div class="feature-item">
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
                <span>Drive Files & Folders</span>
              </div>
              <div class="feature-item">
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
                <span>Contacts & Calendar</span>
              </div>
            </div>
            <button class="btn btn--primary" @click="handleGoogleTakeout">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="currentColor"
                width="18"
                height="18"
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
              Connect with Google
            </button>
          </div>

          <!-- iCloud Option -->
          <div class="migration-card icloud disabled">
            <div class="migration-card-header">
              <div class="migration-card-icon icloud">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <path
                    d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"
                  />
                </svg>
              </div>
              <h3>iCloud</h3>
              <span class="coming-soon-badge">Coming Soon</span>
            </div>
            <p class="migration-card-description">
              Import your photos, files, and data from iCloud. Automatic sync
              and import will be available soon.
            </p>
            <button class="btn btn--primary" disabled>Coming Soon</button>
          </div>

          <!-- Dropbox Option -->
          <div class="migration-card dropbox disabled">
            <div class="migration-card-header">
              <div class="migration-card-icon dropbox">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <path
                    d="M6 1.8L0 5.4l6 3.6 6-3.6-6-3.6zm12 0l-6 3.6 6 3.6 6-3.6-6-3.6zM0 12.6l6 3.6 6-3.6-6-3.6-6 3.6zm18-3.6l-6 3.6 6 3.6 6-3.6-6-3.6zM6 17.4l6 3.6 6-3.6-6-3.6-6 3.6z"
                  />
                </svg>
              </div>
              <h3>Dropbox</h3>
              <span class="coming-soon-badge">Coming Soon</span>
            </div>
            <p class="migration-card-description">
              Import files and folders from Dropbox cloud storage. Automatic
              sync coming soon.
            </p>
            <button class="btn btn--primary" disabled>Coming Soon</button>
          </div>

          <!-- OneDrive Option -->
          <div class="migration-card onedrive disabled">
            <div class="migration-card-header">
              <div class="migration-card-icon onedrive">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <path
                    d="M13.5 7.5c-1.5-2-4-3.5-7-3.5C3.5 4 1 6.5 1 9.5c0 .5.1 1 .2 1.5C.5 11.5 0 12.7 0 14c0 2.2 1.8 4 4 4h15c2.8 0 5-2.2 5-5 0-2.6-2-4.7-4.5-4.9-.3-2.6-2.5-4.6-5-4.6z"
                  />
                </svg>
              </div>
              <h3>OneDrive</h3>
              <span class="coming-soon-badge">Coming Soon</span>
            </div>
            <p class="migration-card-description">
              Import documents and files from Microsoft OneDrive. Integration
              coming soon.
            </p>
            <button class="btn btn--primary" disabled>Coming Soon</button>
          </div>

          <!-- Box Option -->
          <div class="migration-card box disabled">
            <div class="migration-card-header">
              <div class="migration-card-icon box">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <path
                    d="M3 3h18a2 2 0 012 2v14a2 2 0 01-2 2H3a2 2 0 01-2-2V5a2 2 0 012-2zm9 4l-7 4 7 4 7-4-7-4z"
                  />
                </svg>
              </div>
              <h3>Box</h3>
              <span class="coming-soon-badge">Coming Soon</span>
            </div>
            <p class="migration-card-description">
              Import enterprise files and content from Box. Enterprise
              integration coming soon.
            </p>
            <button class="btn btn--primary" disabled>Coming Soon</button>
          </div>
        </div>

        <!-- Upload Section -->
        <div class="upload-section" v-if="showUploadSection">
          <h3>Upload Your Data</h3>
          <div
            class="upload-area"
            @drop="handleDrop"
            @dragover="handleDragOver"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4" />
              <polyline points="17 8 12 3 7 8" />
              <line x1="12" y1="3" x2="12" y2="15" />
            </svg>
            <p>Drag and drop your files here or click to browse</p>
            <input
              type="file"
              multiple
              @change="handleFileSelect"
              ref="fileInput"
              style="display: none"
            />
          </div>
          <button class="btn btn--secondary" @click="() => fileInput?.click()">
            Browse Files
          </button>
          <button
            v-if="selectedFiles.length > 0"
            class="btn btn--primary"
            @click="uploadFiles"
            :disabled="uploading"
          >
            {{ uploading ? 'Uploading...' : 'Upload Files' }}
          </button>
          <div v-if="selectedFiles.length > 0" class="file-list">
            <p class="file-count">
              {{ selectedFiles.length }} file(s) selected
            </p>
            <div class="file-items">
              <div
                v-for="(file, idx) in selectedFiles"
                :key="idx"
                class="file-item"
              >
                {{ file.name }}
              </div>
            </div>
          </div>
          <div
            v-if="uploadMessage"
            :class="['upload-message', uploadMessageType]"
          >
            {{ uploadMessage }}
          </div>
        </div>
      </div>
    </template>
  </LibraryLayout>
</template>

<script setup lang="ts">
import LibraryLayout from '@/components/common/LibraryLayout.vue';
import LibrarySidebar from '@/components/common/LibrarySidebar.vue';
import { ref } from 'vue';
import { useRouter } from 'vue-router';

const router = useRouter();

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
    items: [{ label: 'Data Migration', href: '/data-migration', active: true }],
  },
];

const showUploadSection = ref(false);
const selectedFiles = ref<File[]>([]);
const fileInput = ref<HTMLInputElement>();
const uploading = ref(false);
const uploadMessage = ref('');
const uploadMessageType = ref<'success' | 'error'>('success');

const handleGoogleTakeout = () => {
  router.push('/data-migration/google-takeout');
};

const handleICloud = () => {
  showUploadSection.value = true;
  uploadMessage.value = '';
};

const handleDropbox = () => {
  showUploadSection.value = true;
  uploadMessage.value = '';
};

const handleOneDrive = () => {
  showUploadSection.value = true;
  uploadMessage.value = '';
};

const handleBox = () => {
  showUploadSection.value = true;
  uploadMessage.value = '';
};

const handleDragOver = (e: DragEvent) => {
  e.preventDefault();
  e.stopPropagation();
};

const handleDrop = (e: DragEvent) => {
  e.preventDefault();
  e.stopPropagation();
  const files = Array.from(e.dataTransfer?.files || []);
  selectedFiles.value = files;
};

const handleFileSelect = (e: Event) => {
  const target = e.target as HTMLInputElement;
  const files = Array.from(target.files || []);
  selectedFiles.value = files;
};

const uploadFiles = async () => {
  if (selectedFiles.value.length === 0) return;

  uploading.value = true;
  uploadMessage.value = '';

  try {
    const formData = new FormData();
    selectedFiles.value.forEach((file) => {
      formData.append('files', file);
    });

    const response = await fetch('/api/v1/upload', {
      method: 'POST',
      body: formData,
    });

    if (response.ok) {
      uploadMessage.value = 'Files uploaded successfully!';
      uploadMessageType.value = 'success';
      selectedFiles.value = [];
    } else {
      uploadMessage.value = 'Failed to upload files. Please try again.';
      uploadMessageType.value = 'error';
    }
  } catch (error) {
    uploadMessage.value =
      error instanceof Error
        ? error.message
        : 'An error occurred during upload.';
    uploadMessageType.value = 'error';
  } finally {
    uploading.value = false;
  }
};
</script>

<style lang="scss" scoped>
.migration-content {
  flex: 1;
  overflow-y: auto;
  padding: 0;
  display: flex;
  flex-direction: column;
  max-width: 1400px;
  width: 100%;
  margin: 0 auto;
  padding: $spacing-2xl;
  padding-top: 5rem;
  padding-bottom: $spacing-2xl;
}

.migration-container {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: $spacing-3xl;
  margin-bottom: $spacing-4xl;

  @media (max-width: 768px) {
    grid-template-columns: 1fr;
    gap: $spacing-2xl;
  }
}

.migration-card {
  background: var(--color-gray-800);
  border: 2px solid transparent;
  border-radius: 12px;
  padding: $spacing-3xl;
  margin: 2px;
  display: flex;
  flex-direction: column;
  gap: $spacing-xl;
  transition: all 0.2s ease;

  @media (prefers-color-scheme: light) {
    background: white;
    border-color: var(--color-gray-300);
  }

  &:not(.disabled):hover {
    border-color: var(--color-primary-500);
    transform: translateY(-2px);

    @media (prefers-color-scheme: light) {
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
    }
  }

  &.disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }
}

.migration-card-header {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: $spacing-sm;
  flex-wrap: wrap;

  h3 {
    margin: 0;
    font-size: var(--font-size-xl);
    font-weight: 600;
    color: white;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }
}

.coming-soon-badge {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
  padding: $spacing-xs $spacing-sm;
  border-radius: 6px;
  font-size: var(--font-size-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border: 1px solid rgba(251, 191, 36, 0.3);
}

.migration-card-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  flex-shrink: 0;

  svg {
    width: 24px;
    height: 24px;
  }

  &.google {
    background: rgba(255, 193, 7, 0.15);
    color: #ffc107;
  }

  &.icloud {
    background: rgba(100, 200, 255, 0.15);
    color: #64c8ff;
  }

  &.dropbox {
    background: rgba(0, 97, 255, 0.15);
    color: #0061ff;
  }

  &.onedrive {
    background: rgba(0, 120, 212, 0.15);
    color: #0078d4;
  }

  &.box {
    background: rgba(0, 111, 201, 0.15);
    color: #006fc9;
  }
}

.migration-card-description {
  color: var(--color-gray-400);
  margin: 0;
  font-size: var(--font-size-base);
  line-height: 1.5;

  @media (prefers-color-scheme: light) {
    color: var(--color-gray-600);
  }
}

.migration-card-features {
  display: flex;
  flex-direction: column;
  gap: $spacing-md;

  .feature-item {
    display: flex;
    align-items: center;
    gap: $spacing-sm;
    color: var(--color-gray-400);
    font-size: var(--font-size-sm);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }

    svg {
      flex-shrink: 0;
      color: var(--color-primary-500);
    }

    span {
      line-height: 1.6;
    }
  }
}

.migration-card-steps {
  background: rgba(255, 255, 255, 0.03);
  border-radius: 8px;
  padding: var(--spacing-md) var(--spacing-lg);

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-50);
  }

  ol {
    margin: 0;
    padding-left: 1.25rem;

    li {
      color: var(--color-gray-400);
      margin-bottom: var(--spacing-xs);
      font-size: var(--font-size-sm);
      line-height: 1.6;

      @media (prefers-color-scheme: light) {
        color: var(--color-gray-600);
      }

      &:last-child {
        margin-bottom: 0;
      }
    }
  }
}

.upload-section {
  margin-top: $spacing-4xl;
  padding: $spacing-3xl;
  background: var(--color-gray-900);
  border: 1px solid var(--color-gray-800);
  border-radius: var(--border-radius-lg);

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-100);
    border-color: var(--color-gray-200);
  }

  h3 {
    margin: 0 0 $spacing-xl 0;
    color: white;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }
}

.upload-area {
  border: 2px dashed var(--color-gray-700);
  border-radius: var(--border-radius-lg);
  padding: var(--spacing-3xl) var(--spacing-2xl);
  text-align: center;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--spacing-md);
  margin-bottom: var(--spacing-lg);

  @media (prefers-color-scheme: light) {
    border-color: var(--color-gray-300);
  }

  &:hover {
    border-color: var(--color-primary-600);
    background: rgba(37, 99, 235, 0.05);
  }

  svg {
    width: 48px;
    height: 48px;
    color: var(--color-primary-600);
    opacity: 0.7;
  }

  p {
    margin: 0;
    color: var(--color-gray-400);

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-600);
    }
  }
}

.file-list {
  margin-top: var(--spacing-lg);
  padding: var(--spacing-lg);
  background: rgba(255, 255, 255, 0.05);
  border-radius: var(--border-radius);

  @media (prefers-color-scheme: light) {
    background: rgba(0, 0, 0, 0.02);
  }

  .file-count {
    margin: 0 0 var(--spacing-md) 0;
    font-weight: 500;
    color: white;

    @media (prefers-color-scheme: light) {
      color: var(--color-gray-900);
    }
  }
}

.file-items {
  max-height: 200px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.file-item {
  padding: var(--spacing-sm);
  background: rgba(255, 255, 255, 0.1);
  border-radius: var(--border-radius);
  font-size: var(--font-size-sm);
  color: var(--color-gray-300);
  word-break: break-all;

  @media (prefers-color-scheme: light) {
    background: rgba(0, 0, 0, 0.05);
    color: var(--color-gray-700);
  }
}

.upload-message {
  margin-top: var(--spacing-lg);
  padding: var(--spacing-lg);
  border-radius: var(--border-radius);
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
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  padding: $spacing-md $spacing-lg;
  border-radius: var(--border-radius-lg);
  font-size: var(--font-size-base);
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  border: none;
  text-decoration: none;

  svg {
    flex-shrink: 0;
  }

  &--primary {
    background: linear-gradient(
      135deg,
      var(--color-primary-600) 0%,
      var(--color-primary-700) 100%
    );
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
  }

  &--secondary {
    background: transparent;
    color: var(--color-primary-600);
    border: 1px solid var(--color-primary-600);

    &:hover {
      background: rgba(37, 99, 235, 0.1);
    }

    @media (prefers-color-scheme: light) {
      color: var(--color-primary-600);
      border-color: var(--color-primary-600);
    }
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
