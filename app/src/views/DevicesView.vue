<template>
  <div class="landing-main">
    <GradientOverlays />
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
              :key="device.device_path"
              :device="device"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'

import DeviceCard from '../components/DeviceCard.vue'
import GradientOverlays from '@/components/home/GradientOverlays.vue'
import type { Summary } from '@/types/summary'
import type { Device } from '@/types/device'

const devices = ref<Device[]>([])
const summary = ref<Summary>({
  total_devices: 0,
  total_bytes: 0,
  used_bytes: 0,
  avail_bytes: 0,
  total_tb: 0,
  used_tb: 0,
  avail_tb: 0,
})
const loading = ref(false)

const fetchDevices = async () => {
  loading.value = true
  try {
    const res = await fetch('/api/v1/storage/devices/status')
    const data = await res.json()
    devices.value = data.devices || []
    summary.value = calculateSummary(devices.value)
  } finally {
    loading.value = false
  }
}

const calculateSummary = (devices: Device[]): Summary => {
  let total_bytes = 0,
    used_bytes = 0,
    avail_bytes = 0
  for (const d of devices) {
    total_bytes += d.total_bytes
    used_bytes += d.used_bytes
    avail_bytes += d.avail_bytes
  }
  return {
    total_devices: devices.length,
    total_bytes,
    used_bytes,
    avail_bytes,
    total_tb: total_bytes / 1e12,
    used_tb: used_bytes / 1e12,
    avail_tb: avail_bytes / 1e12,
  }
}

onMounted(fetchDevices)
</script>

<style lang="scss" scoped>
.landing-main {
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
  font-size: $font-size-2xl;
  font-weight: 700;
  margin: 0;
  color: $color-gray-900;
}
@media (prefers-color-scheme: dark) {
  .devices-title {
    color: white;
  }
}
.devices-subtitle {
  color: $color-gray-500;
  font-size: 1rem;
  margin-top: 0.5rem;
}
.devices-refresh-button {
  background: $color-gray-100;
  color: $color-gray-900;
  border: 1px solid $color-gray-300;
  border-radius: 8px;
  padding: 0.5rem 1.2rem;
  font-size: 1rem;
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
  color: $color-gray-500;
  font-size: 1.2rem;
  margin-top: 4rem;
}

.storage-bar-card {
  width: 100%;
  margin-top: 1rem;
  background: $color-gray-100;
  border-radius: 12px;
  padding: 1.5rem 2rem;
  box-shadow: 0 2px 8px 0 rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.storage-bar {
  display: flex;
  height: 32px;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.03);
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
  font-size: 1rem;
  margin-top: 0.5rem;
}
.storage-bar-total {
  color: $color-gray-900;
  font-weight: 600;
}
.storage-bar-used {
  color: $color-red-400;
}
.storage-bar-free {
  color: $color-green-400;
}
</style>
