<template>
  <div class="page-container">
    <main class="landing-main">
      <div class="landing-container">
        <GradientOverlays />
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
                <svg v-if="loading" class="refresh-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="23 4 23 10 17 10"></polyline>
                  <polyline points="1 20 1 14 7 14"></polyline>
                  <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
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
              <DeviceCard
                v-for="device in devices"
                :key="device.device_path"
                :device="device"
              />
            </div>
            <div v-if="devices.length > 0" class="devices-total">
              <div class="devices-total-icon">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <path d="M12 6v6l4 2"></path>
                </svg>
              </div>
              <div class="devices-total-info">
                <h3 class="devices-total-title">Total Capacity</h3>
                <p class="devices-total-capacity">{{ summary.total_tb.toFixed(2) }} TB</p>
                <p class="devices-total-used">
                  {{ summary.used_tb.toFixed(2) }} TB used • {{ summary.avail_tb.toFixed(2) }} TB free
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>


<script setup lang="ts">
import { ref, onMounted } from 'vue';
import DeviceCard from '../components/DeviceCard.vue';
import TopNav from '@/components/home/TopNav.vue';
import GradientOverlays from '@/components/home/GradientOverlays.vue';

interface Device {
  name: string;
  type: string;
  device_path: string;
  mount_point: string;
  file_system: string;
  total_bytes: number;
  used_bytes: number;
  avail_bytes: number;
  percent_used: number;
  is_internal: boolean;
  is_removable: boolean;
  is_read_only: boolean;
  model: string;
  categories: Record<string, number>;
}

interface Summary {
  total_devices: number;
  total_bytes: number;
  used_bytes: number;
  avail_bytes: number;
  total_tb: number;
  used_tb: number;
  avail_tb: number;
}

const navLinks = [
  { name: 'Cirrus', href: '/cirrus' },
  { name: 'Photos', href: '/photos' },
  { name: 'Books', href: '/books' },
];

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

async function fetchDevices() {
  loading.value = true;
  try {
    const res = await fetch('/api/v1/storage/devices/status');
    const data = await res.json();
    devices.value = data.devices || [];
    summary.value = calculateSummary(devices.value);
  } finally {
    loading.value = false;
  }
}

function calculateSummary(devices: Device[]): Summary {
  let total_bytes = 0, used_bytes = 0, avail_bytes = 0;
  for (const d of devices) {
    total_bytes += d.total_bytes;
    used_bytes += d.used_bytes;
    avail_bytes += d.avail_bytes;
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
}

onMounted(fetchDevices);
</script>

<style scoped>
</style>
