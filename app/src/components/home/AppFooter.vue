<template>
  <footer class="app-footer">
    <div class="footer-content">
      <span class="footer-text-container">
        <a
          href="https://github.com/autobutler-org/autobutler"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p class="footer-text">Autobutler LLC.</p>
          <img src="/img/github/github-mark-white.svg" alt="GitHub logo" />
        </a>
      </span>

      <div class="storage-bar-container">
        <div v-if="storageData" class="storage-info">
          <span class="storage-label">Storage</span>
          <span class="storage-stats">
            {{ formatBytes(storageData.usedBytes) }} of
            {{ formatBytes(storageData.totalBytes) }} used
          </span>
        </div>
        <div v-if="storageData" class="storage-bar">
          <div
            class="storage-bar-fill"
            :style="{ width: usagePercentage + '%' }"
          ></div>
        </div>
        <div v-if="!storageData && !isLoading" class="storage-error">
          Storage information unavailable
        </div>
        <div v-if="isLoading" class="storage-loading">
          Loading storage data...
        </div>
      </div>

      <a
        :href="swaggerLink"
        target="_blank"
        rel="noopener noreferrer"
        class="swagger-link"
      >
        <img src="/favicons/swagger.png" alt="Swagger logo" />
        Interactive API Docs
      </a>
    </div>
  </footer>
</template>

<script lang="ts" setup>
import DevicesService from '@/services/devicesService';
import type { Device } from '@/types/device';
import { computed, onMounted, ref } from 'vue';

const storageData = ref<{
  totalBytes: number;
  usedBytes: number;
  availableBytes: number;
} | null>(null);

const isLoading = ref(true);

const usagePercentage = computed(() => {
  if (!storageData.value || storageData.value.totalBytes === 0) return 0;
  return Math.round(
    (storageData.value.usedBytes / storageData.value.totalBytes) * 100,
  );
});

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return Math.round((bytes / Math.pow(k, i)) * 10) / 10 + ' ' + sizes[i];
};

const loadStorageData = async () => {
  try {
    isLoading.value = true;
    const response = await DevicesService.getDeviceStatuses();

    // Aggregate storage from all devices
    const aggregated = response.devices.reduce(
      (acc, device: Device) => {
        return {
          totalBytes: acc.totalBytes + device.totalBytes,
          usedBytes: acc.usedBytes + device.usedBytes,
          availableBytes: acc.availableBytes + device.availableBytes,
        };
      },
      { totalBytes: 0, usedBytes: 0, availableBytes: 0 },
    );

    storageData.value = aggregated;
  } catch (error) {
    console.error('Failed to load storage data:', error);
  } finally {
    isLoading.value = false;
  }
};

const swaggerLink =
  import.meta.env.MODE === 'development'
    ? 'http://localhost:8080/swagger'
    : '/swagger';

onMounted(() => {
  loadStorageData();
});
</script>

<style lang="scss" scoped>
.app-footer {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: $spacing-md $spacing-lg;
  color: $theme-palette-text-muted;
  background: hsl(from $theme-palette-bg-nav h s l / 0.95);
  backdrop-filter: blur(1.25rem);
  border-top: 0.0625rem solid hsl(from $theme-palette-bg-primary h s l / 0.1);
  width: 100%;
  margin-top: auto;
  flex-shrink: 0;
}

.footer-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  max-width: 100%;
  gap: $spacing-lg;
  flex-wrap: nowrap;

  @media (max-width: 48rem) {
    gap: $spacing-sm;
  }
}

.footer-text-container {
  flex-shrink: 0;
  margin-right: auto;

  .footer-text {
    font-size: $theme-font-size-xs;
    margin: 0;
  }

  img {
    width: 1.25rem;
    height: 1.25rem;
    opacity: 0.7;
    transition: opacity 0.2s ease;
  }

  a {
    text-decoration: none;
    color: $theme-palette-text-secondary;

    display: flex;
    align-items: center;
    gap: $spacing-xs;

    &:hover {
      & img {
        opacity: 1;
      }
      text-decoration: underline;
    }
  }
}

.storage-bar-container {
  flex: 1 1 22.5rem;
  max-width: 32.5rem;
  min-width: 15rem;
  display: flex;
  flex-direction: column;
  gap: calc($spacing-xs / 2);
  padding: $spacing-xs $spacing-sm;
  border-radius: $border-radius-md;
  background: hsl(from $theme-palette-bg-primary h s l / 0.2);
  border: 0.0625rem solid hsl(from $theme-palette-bg-primary h s l / 0.35);

  @media (max-width: 48rem) {
    min-width: 0;
    max-width: 100%;
    width: 100%;
  }
}

.storage-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: $theme-font-size-xs;
  gap: $spacing-xs;
  line-height: 1.2;

  .storage-stats {
    text-align: right;
    white-space: nowrap;
  }

  @media (max-width: 30rem) {
    align-items: center;
    justify-content: center;
    gap: 0.125rem;

    .storage-stats {
      text-align: center;
      white-space: normal;
    }
  }
}

.storage-label {
  font-weight: 600;
  color: $theme-palette-text-secondary;
}

.storage-stats {
  color: $theme-palette-text-muted;
}

.storage-bar {
  width: 100%;
  align-self: stretch;
  height: 0.5rem;
  background: hsl(from $theme-palette-bg-primary h s l / 0.3);
  border-radius: 0.25rem;
  overflow: hidden;
  position: relative;
}

.storage-bar-fill {
  height: 100%;
  background: linear-gradient(
    90deg,
    $theme-palette-accent 0%,
    $theme-palette-accent-hover 100%
  );
  transition: width 0.3s ease;
  border-radius: 0.25rem;
}

.storage-loading,
.storage-error {
  font-size: $theme-font-size-xs;
  color: $theme-palette-text-muted;
  text-align: center;
  padding: $spacing-sm;
}

.storage-error {
  color: $theme-palette-text-secondary;
  opacity: 0.7;
}

.swagger-link {
  // Same as other links, like top nav
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  font-size: $theme-font-size-xs;
  color: $theme-palette-text-muted;
  text-decoration: none;
  transition: color 0.2s ease;
  margin-left: auto;

  img {
    width: 1.25rem;
    height: 1.25rem;
  }

  &:hover {
    color: $theme-palette-text-primary;
    text-decoration: underline;
  }
}
</style>
