<template>
  <footer class="app-footer">
    <div class="footer-content">
      <div class="footer-left">
        <p class="footer-text">Autobutler LLC.</p>
        <a
          href="https://github.com/autobutler-org/autobutler"
          target="_blank"
          rel="noopener noreferrer"
        >
          <img src="/img/github/github-mark-white.svg" alt="GitHub logo" />
        </a>
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

      <div class="storage-bar-container">
        <div v-if="storageData" class="storage-info">
          <span class="storage-label">Total Storage</span>
          <span class="storage-stats">
            {{ formatBytes(storageData.usedBytes) }} of
            {{ formatBytes(storageData.totalBytes) }} used
            <span class="storage-available"
              >({{ formatBytes(storageData.availableBytes) }} available)</span
            >
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
  backdrop-filter: blur(20px);
  border-top: 1px solid hsl(from $theme-palette-bg-primary h s l / 0.1);
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
  flex-wrap: wrap;

  @media (max-width: 768px) {
    flex-direction: column;
    gap: $spacing-md;
  }
}

.footer-left {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  flex-shrink: 0;

  img {
    width: 20px;
    height: 20px;
    opacity: 0.7;
    transition: opacity 0.2s ease;
  }

  a:hover img {
    opacity: 1;
  }
}

.footer-text {
  font-size: $theme-font-size-xs;
  margin: 0;
}

.storage-bar-container {
  flex: 1;
  max-width: 400px;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;

  @media (max-width: 768px) {
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

  @media (max-width: 480px) {
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
  }
}

.storage-label {
  font-weight: 600;
  color: $theme-palette-text-secondary;
}

.storage-stats {
  color: $theme-palette-text-muted;
}

.storage-available {
  color: $theme-palette-text-muted;
  opacity: 0.8;
}

.storage-bar {
  height: 8px;
  background: hsl(from $theme-palette-bg-primary h s l / 0.3);
  border-radius: 4px;
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
  border-radius: 4px;
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
  font-size: $theme-font-size-sm;
  color: $theme-palette-text-muted;
  text-decoration: none;
  transition: color 0.2s ease;

  &:hover {
    color: $theme-palette-text-primary;
    text-decoration: underline;
  }
}
</style>
