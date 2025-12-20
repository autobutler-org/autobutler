
<template>
  <div class="site-main-bg">
    <main class="landing-main">
      <GradientOverlays />
      <div class="landing-container">
        <TopNav :navLinks="navLinks" />
        <div class="devices-page">
          <div class="devices-header">
            <div class="devices-header-content">
              <div>
                <h1 class="devices-title">Storage Devices</h1>
                <p class="devices-subtitle">
                  Monitor capacity, usage, and content categories across all connected drives
                </p>
              </div>
              <button
                class="devices-refresh-button"
                :disabled="loading"
                @click="fetchDevices"
                title="Refresh storage devices"
              >
                <svg
                  v-if="loading"
                  class="refresh-icon"
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <polyline points="23 4 23 10 17 10"></polyline>
                  <polyline points="1 20 1 14 7 14"></polyline>
                  <path
                    d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"
                  ></path>
                </svg>
                <span>Refresh</span>
              </button>
            </div>
          </div>
          <div id="devices-content">
            <div v-if="devices.length === 0 && !loading" class="devices-empty">
              <p>No storage devices detected</p>
            </div>
            <div v-else class="devices-grid">
              <DeviceCard v-for="device in devices" :key="device.device_path" :device="device" />
            </div>
            <div v-if="devices.length > 0" class="devices-total">
              <div class="devices-total-info">
                <h3 class="devices-total-title">Storage Breakdown</h3>
                <div class="storage-bar-card">
                  <div class="storage-bar">
                    <div
                      v-for="cat in storageCategories"
                      :key="cat.name"
                      class="storage-bar-section"
                      :style="{
                        width: cat.percent + '%',
                        background: cat.color
                      }"
                      :title="cat.name + ': ' + cat.sizeDisplay"
                    ></div>
                  </div>
                  <div class="storage-bar-legend">
                    <span v-for="cat in storageCategories" :key="cat.name" class="storage-bar-legend-item">
                      <span class="legend-dot" :style="{ background: cat.color }"></span>
                      {{ cat.name }} ({{ cat.sizeDisplay }})
                    </span>
                  </div>
                  <div class="storage-bar-summary">
                    <span class="storage-bar-total">Total: {{ summary.total_tb.toFixed(2) }} TB</span>
                    <span class="storage-bar-used">Used: {{ summary.used_tb.toFixed(2) }} TB</span>
                    <span class="storage-bar-free">Free: {{ summary.avail_tb.toFixed(2) }} TB</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
// ...existing code...

// Compute storage categories for the bar
const storageCategories = computed(() => {
  // Demo/mock values for UI only
  // Replace with real aggregation from devices/categories if available
  const cats = [
    { name: 'Documents', size_tb: 120, color: 'var(--color-red-400)' },
    { name: 'Applications', size_tb: 80, color: 'var(--color-orange-400)' },
    { name: 'Messages', size_tb: 60, color: 'var(--color-yellow-400)' },
    { name: 'Trash', size_tb: 10, color: 'var(--color-green-400)' },
    { name: 'System Data', size_tb: 100, color: 'var(--color-gray-400)' },
    { name: 'Free', size_tb: summary.value.avail_tb, color: 'var(--color-gray-200)' }
  ];
  const total = summary.value.total_tb || 370 + summary.value.avail_tb;
  return cats.map(cat => ({
    ...cat,
    percent: total ? (cat.size_tb / total * 100) : 0,
    sizeDisplay: cat.size_tb.toFixed(2) + ' TB'
  })).filter(cat => cat.size_tb > 0);
});
import DeviceCard from '../components/DeviceCard.vue'
import TopNav from '@/components/home/TopNav.vue'
import GradientOverlays from '@/components/home/GradientOverlays.vue'

interface Device {
  name: string
  type: string
  device_path: string
  mount_point: string
  file_system: string
  total_bytes: number
  used_bytes: number
  avail_bytes: number
  percent_used: number
  is_internal: boolean
  is_removable: boolean
  is_read_only: boolean
  model: string
  categories: Record<string, number>
}

interface Summary {
  total_devices: number
  total_bytes: number
  used_bytes: number
  avail_bytes: number
  total_tb: number
  used_tb: number
  avail_tb: number
}

const navLinks = [
  { name: 'Cirrus', href: '/cirrus' },
  { name: 'Photos', href: '/photos' },
  { name: 'Books', href: '/books' },
]

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

async function fetchDevices() {
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

function calculateSummary(devices: Device[]): Summary {
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

/* Use the same layout and style as HomeView and BooksView */
.site-main-bg {
  height: 100vh;
  min-height: 100vh;
  width: 100vw;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: linear-gradient(180deg, hsl(225, 25%, 15%) 0%, hsl(225, 30%, 10%) 100%);
}
@media (prefers-color-scheme: light) {
  .site-main-bg {
    background: linear-gradient(180deg, hsl(225, 15%, 95%) 0%, hsl(225, 20%, 98%) 100%);
  }
}
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
  padding: 0 var(--spacing-2xl);
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
  padding-bottom: var(--spacing-2xl);
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
  font-size: var(--font-size-2xl);
  font-weight: 700;
  margin: 0;
  color: var(--color-gray-900);
}
@media (prefers-color-scheme: dark) {
  .devices-title {
    color: white;
  }
}
.devices-subtitle {
  color: var(--color-gray-500);
  font-size: 1rem;
  margin-top: 0.5rem;
}
.devices-refresh-button {
  background: var(--color-gray-100);
  color: var(--color-gray-900);
  border: 1px solid var(--color-gray-300);
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
  gap: var(--spacing-xl);
  margin-top: var(--spacing-xl);
}
.devices-empty {
  text-align: center;
  color: var(--color-gray-500);
  font-size: 1.2rem;
  margin-top: 4rem;
}

.storage-bar-card {
  width: 100%;
  margin-top: 1rem;
  background: var(--color-gray-100);
  border-radius: 12px;
  padding: 1.5rem 2rem;
  box-shadow: 0 2px 8px 0 rgba(0,0,0,0.04);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.storage-bar {
  display: flex;
  height: 32px;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 4px 0 rgba(0,0,0,0.03);
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
  color: var(--color-gray-900);
  font-weight: 600;
}
.storage-bar-used {
  color: var(--color-red-400);
}
.storage-bar-free {
  color: var(--color-green-400);
}
</style>
