<template>
  <div class="view-content">
    <div class="landing-container">
      <div class="devices-page">
        <div class="devices-header">
          <div class="devices-header-content">
            <div>
              <h1 class="devices-title">Storage Devices</h1>
              <p class="devices-subtitle">
                Monitor capacity, usage, and content categories across all
                connected drives
              </p>
            </div>
            <button
              class="devices-refresh-button"
              :disabled="loading"
              @click="fetchDevices"
              title="Refresh storage devices"
            >
              <span>Refresh</span>
            </button>
          </div>
        </div>
        <div id="devices-content">
          <div v-if="devices.length === 0 && !loading" class="devices-empty">
            <p>No storage devices detected</p>
          </div>
          <div v-else class="devices-grid">
            <DeviceCard
              v-for="device in devices"
              :key="device.devicePath"
              :device="device"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import DevicesService from '@/services/devicesService';
import { useDeviceNotificationStore } from '@/stores/deviceNotifications';
import type { Device } from '@/types/device';
import type { Summary } from '@/types/summary';
import DeviceCard from '../components/DeviceCard.vue';

const devices = ref<Device[]>([]);
const summary = ref<Summary>({
  total_devices: 0,
  total_bytes: 0,
  used_bytes: 0,
  avail_bytes: 0,
  total_tb: 0,
  used_tb: 0,
  avail_tb: 0,
});
const loading = ref(false);
const deviceNotificationStore = useDeviceNotificationStore();

const fetchDevices = async () => {
  loading.value = true;
  try {
    const data = await DevicesService.getDeviceStatuses();
    devices.value = data.devices || [];
    summary.value = calculateSummary(devices.value);
  } finally {
    loading.value = false;
  }
};

const calculateSummary = (devices: Device[]): Summary => {
  let total_bytes = 0,
    used_bytes = 0,
    avail_bytes = 0;
  for (const d of devices) {
    total_bytes += d.totalBytes;
    used_bytes += d.usedBytes;
    avail_bytes += d.availableBytes;
  }
  return {
    total_devices: devices.length,
    total_bytes,
    used_bytes,
    avail_bytes,
    total_tb: total_bytes / 1e12,
    used_tb: used_bytes / 1e12,
    avail_tb: avail_bytes / 1e12,
  };
};

onMounted(() => {
  fetchDevices();
  // Mark devices as viewed when page is opened
  deviceNotificationStore.markDevicesAsViewed();
});
</script>

<style lang="scss" scoped>
.view-content {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  position: relative;
}
.landing-container {
  display: flex;
  flex-direction: column;
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
  padding: 0 $spacing-2xl;
}
.devices-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
#devices-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-bottom: $spacing-2xl;
}
.devices-header {
  padding: 2rem 0 1rem 0;
}
.devices-header-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 2rem;
}
.devices-title {
  font-size: $theme-font-size-2xl;
  font-weight: 700;
  margin: 0;
  color: $theme-palette-text-primary;
}
.devices-subtitle {
  color: $theme-palette-text-muted;
  font-size: $theme-font-size-base;
  margin-top: 0.5rem;
}
.devices-refresh-button {
  background: $theme-palette-bg-secondary;
  color: $theme-palette-text-primary;
  border: 1px solid $theme-palette-border;
  border-radius: 8px;
  padding: 0.5rem 1.2rem;
  font-size: $theme-font-size-base;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  transition: background 0.2s;
}
.devices-refresh-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.devices-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: $spacing-xl;
  margin-top: $spacing-xl;
}
.devices-empty {
  text-align: center;
  color: $theme-palette-text-muted;
  font-size: 1.2rem;
  margin-top: 4rem;
}

.storage-bar-card {
  width: 100%;
  margin-top: 1rem;
  background: $theme-palette-bg-secondary;
  border-radius: 12px;
  padding: 1.5rem 2rem;
  box-shadow: 0 2px 8px 0 hsl(from $theme-palette-border h s l / 0.04);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.storage-bar {
  display: flex;
  height: 32px;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 4px 0 hsl(from $theme-palette-border h s l / 0.02);
}
.storage-bar-section {
  height: 100%;
  transition: width 0.3s;
}
.storage-bar-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 1.2rem;
  font-size: 0.95rem;
  margin-top: 0.5rem;
}
.storage-bar-legend-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.legend-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  display: inline-block;
}
.storage-bar-summary {
  display: flex;
  gap: 2rem;
  font-size: $theme-font-size-base;
  margin-top: 0.5rem;
}
.storage-bar-total {
  color: $theme-palette-text-primary;
  font-weight: 600;
}
.storage-bar-used {
  color: #f87171;
}
.storage-bar-free {
  color: #4ade80;
}
</style>
