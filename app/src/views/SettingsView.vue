<template>
  <LibraryLayout>
    <template #sidebar>
      <LibrarySidebar :sections="sidebarSections" />
    </template>
    <template #title>
      <h2 class="library-title">Settings & Metrics</h2>
    </template>
    <template #subtitle>
      <div class="library-subtitle">
        <div class="settings-header-actions">
          <button aria-label="Save">
            <SaveIcon />
          </button>
          <button aria-label="Refresh" @click="fetchDevices">
            <RefreshIcon />
          </button>
        </div>
      </div>
    </template>
    <template #main>
      <div class="settings-content redesigned">
        <div class="settings-cards-row">
          <section id="opentelemetry" class="settings-section settings-card">
            <div class="settings-section-header">
              <OpenTelemetryIcon />
              <h2>OpenTelemetry</h2>
            </div>
            <div class="settings-section-toolbar">
              <button>Configuration</button>
              <button>Metrics</button>
              <button>Traces</button>
              <button>Logs</button>
              <button>
                <SearchIcon />
                Search settings
              </button>
            </div>
            <div class="settings-section-card mock-card">
              <span class="mock-badge">mock</span>
              <h3>Collector & Exporter</h3>
              <p>Configure where telemetry is sent and how it's batched</p>
            </div>
          </section>
          <section id="storage" class="settings-section settings-card">
            <div class="settings-section-header">
              <h2>Storage & Backups</h2>
            </div>
            <div class="settings-section-card">
              <p class="settings-section-description">
                Mark a USB device as a backup device. When set, files on that
                device will be treated as backups and protected from accidental
                deletion.
              </p>
              <ul class="device-list">
                <li
                  v-for="d in devicesForBackup"
                  :key="d.usbInfo?.serial || d.devicePath"
                  class="device-list-item"
                >
                  <div class="device-name">{{ d.name || d.devicePath }}</div>
                  <div>
                    <!-- Button to do a backup to the device -->
                    <button
                      class="backup-action-button"
                      @click="backupToDevice(d)"
                    >
                      Backup to device
                    </button>
                  </div>
                </li>
              </ul>
            </div>
          </section>
        </div>
        <section class="settings-metrics-section settings-card">
          <span class="mock-badge">mock</span>
          <h3>OpenTelemetry Metrics</h3>
          <p>Live instrumentation snapshots for the NAS and file browser.</p>
          <div class="settings-metrics-grid redesigned-metrics">
            <div class="settings-metric-card">
              <h4>CPU Usage (%)</h4>
              <span class="settings-metric-time">Last 15m</span>
              <div class="settings-metric-chart">Line chart placeholder</div>
              <div class="settings-metric-filters">
                <button>System</button>
                <button>Collector</button>
              </div>
              <table class="settings-metric-stats compact">
                <tbody>
                  <tr>
                    <td>Current</td>
                    <td>32%</td>
                  </tr>
                  <tr>
                    <td>Peak</td>
                    <td>77%</td>
                  </tr>
                  <tr>
                    <td>Avg</td>
                    <td>28%</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="settings-metric-card">
              <h4>Memory Usage (GB)</h4>
              <span class="settings-metric-time">Last 15m</span>
              <div class="settings-metric-chart">Area chart placeholder</div>
              <div class="settings-metric-filters">
                <button>Total</button>
                <button>Used</button>
                <button>Collector</button>
              </div>
              <table class="settings-metric-stats compact">
                <tbody>
                  <tr>
                    <td>Used</td>
                    <td>18.2 GB</td>
                  </tr>
                  <tr>
                    <td>Cache</td>
                    <td>2.9 GB</td>
                  </tr>
                  <tr>
                    <td>Free</td>
                    <td>13.6 GB</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="settings-metric-card">
              <h4>Disk I/O (MB/s)</h4>
              <span class="settings-metric-time">Last 1h</span>
              <div class="settings-metric-chart">Bar chart placeholder</div>
              <div class="settings-metric-filters">
                <button>Read</button>
                <button>Write</button>
              </div>
              <table class="settings-metric-stats compact">
                <tbody>
                  <tr>
                    <td>Read</td>
                    <td>210 MB/s</td>
                  </tr>
                  <tr>
                    <td>Write</td>
                    <td>185 MB/s</td>
                  </tr>
                  <tr>
                    <td>Latency</td>
                    <td>4.6 ms</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="settings-metric-card">
              <h4>File Ops Rate (ops/s)</h4>
              <span class="settings-metric-time">Last 1h</span>
              <div class="settings-metric-chart">Line chart placeholder</div>
              <div class="settings-metric-filters">
                <button>Copy</button>
                <button>Move</button>
                <button>Delete</button>
              </div>
              <table class="settings-metric-stats compact">
                <tbody>
                  <tr>
                    <td>Current</td>
                    <td>42 ops/s</td>
                  </tr>
                  <tr>
                    <td>Peak</td>
                    <td>120 ops/s</td>
                  </tr>
                  <tr>
                    <td>Errors</td>
                    <td>0.2%</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <h3>Recent Telemetry Events</h3>
          <p>Latest exporter status, dropped spans, and warnings.</p>
          <table class="settings-events-table compact">
            <thead>
              <tr>
                <th>Time</th>
                <th>Message</th>
                <th>Component</th>
                <th>Level</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>2025-04-12 10:14:03</td>
                <td>Exporter connected to http://otel-collector:4317</td>
                <td>Exporter</td>
                <td>Info</td>
              </tr>
              <tr>
                <td>2025-04-12 10:10:22</td>
                <td>Batch size adjusted to 512 KB</td>
                <td>Collector</td>
                <td>Info</td>
              </tr>
              <tr>
                <td>2025-04-12 09:58:07</td>
                <td>Transient failure, retry scheduled</td>
                <td>Exporter</td>
                <td>Warning</td>
              </tr>
            </tbody>
          </table>
          <div class="settings-events-actions">
            <button>Clear</button>
            <button>Download</button>
          </div>
        </section>
      </div>
    </template>
  </LibraryLayout>
</template>

<script lang="ts" setup>
import LibraryLayout from '@/components/common/LibraryLayout.vue';
import LibrarySidebar from '@/components/common/LibrarySidebar.vue';
import OpenTelemetryIcon from '@/components/icons/OpenTelemetryIcon.vue';
import RefreshIcon from '@/components/icons/RefreshIcon.vue';
import SaveIcon from '@/components/icons/SaveIcon.vue';
import SearchIcon from '@/components/icons/SearchIcon.vue';
import DevicesService from '@/services/devicesService';
import { useCirrusDeviceStore } from '@/stores/cirrusDeviceStore';
import type { Device } from '@/types/device';
import { computed, onMounted, ref } from 'vue';

const sidebarSections = [
  {
    title: 'Sections',
    items: [
      { label: 'General', active: true },
      { label: 'Users & Access' },
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
];

const deviceStore = useCirrusDeviceStore();
const devices = ref<Device[]>([]);
const devicesForBackup = computed<Device[]>(() =>
  devices.value.filter((d) => d.usbInfo && d.isEnabled),
);

const fetchDevices = async () => {
  try {
    const resp = await DevicesService.getDeviceStatuses();
    devices.value = resp.devices || [];
    deviceStore.setDevices(resp.devices || []);
  } catch {
    devices.value = [];
    deviceStore.setDevices([]);
  }
};

onMounted(() => {
  fetchDevices();
});

const backupToDevice = async (d: Device) => {
  if (
    !window.confirm(
      `Backup data to device ${d.name || d.devicePath}? This will copy all files to the device.`,
    )
  ) {
    return;
  }
  // Persist change to backend and update local state on success
  try {
    const serial = d.usbInfo?.serial || '';
    // Currently, sourceDeviceSerial is empty string to denote the primary storage device
    await DevicesService.backupToDevice('', serial);
  } catch (e) {
    console.error('Failed to start backup', e);
  }
};
</script>

<style lang="scss" scoped>
.device-link {
  color: $theme-palette-text-primary;
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s ease;
  padding: 0;
  background: none;
  height: 2rem;
  display: flex;
  align-items: center;

  &:hover {
    color: $theme-palette-accent;
    text-decoration: underline;
  }

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-primary;
    &:hover {
      color: $theme-palette-accent;
    }
  }
}
.device-list {
  padding-left: 0;
}
.device-list-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  width: 100%;
}
.device-name {
  flex: 1 1 auto;
  float: left;
  text-align: left;
  font-size: $theme-font-size-sm;
  font-weight: 500;
}
.settings-sidebar-thanks {
  margin-top: $spacing-2xl;
  padding: $spacing-md;
  border-top: 1px solid $theme-palette-border;
}
.settings-thanks-link {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  color: $theme-palette-accent;
  text-decoration: none;
  font-weight: 600;
  margin-top: $spacing-xs;
}
.settings-thanks-link:hover {
  text-decoration: underline;
}
.library-sidebar {
  padding-top: $spacing-xl;
}
.library-sidebar-section {
  margin-bottom: $spacing-xl;
}
.library-sidebar-title {
  font-size: $theme-font-size-xs;
  font-weight: 700;
  text-transform: uppercase;
  color: $theme-palette-text-muted;
  margin-bottom: $spacing-xs;
  letter-spacing: 0.05em;
}
.library-sidebar-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
}
.library-sidebar-item {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius;
  color: $theme-palette-text-secondary;
  text-decoration: none;
  transition: all 0.2s ease;
  font-size: $theme-font-size-sm;
  cursor: pointer;
}
.library-sidebar-item:hover {
  background: $theme-palette-bg-secondary;
  color: $theme-palette-text-primary;
}
.library-sidebar-item.active {
  background: $theme-palette-accent;
  color: $theme-palette-text-inverse;
}
.library-sidebar-count {
  margin-left: auto;
  font-size: $theme-font-size-xs;
  color: $theme-palette-text-muted;
}
.library-sidebar-item.active .library-sidebar-count {
  color: $theme-palette-accent-hover;
}
.settings-header {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-lg 0 $spacing-md 0;
  border-bottom: 1px solid $theme-palette-border;
  margin-bottom: $spacing-xl;
}
.settings-title {
  font-size: $theme-font-size-3xl;
  font-weight: 700;
  margin: 0;
  color: $theme-palette-text-primary;
}
.settings-header-actions {
  display: flex;
  gap: $spacing-md;
}
.settings-section {
  margin-bottom: $spacing-2xl;
}
.settings-section-header {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  margin-bottom: $spacing-xs;
}
.settings-section-header h2 {
  font-size: $theme-font-size-xl;
  font-weight: 600;
  margin: 0;
  color: $theme-palette-text-primary;
}
.settings-section-description {
  margin-bottom: $spacing-md;
  color: $theme-palette-text-muted;
}
.settings-section-card {
  background: $theme-palette-bg-primary;
  border-radius: $border-radius;
  box-shadow: $shadow-sm;
  padding: $spacing-xl;
  color: $theme-palette-text-primary;
  border: 1px solid $theme-palette-border;
  margin-bottom: $spacing-md;
}
.settings-section-toolbar {
  display: flex;
  gap: $spacing-xs;
  margin-bottom: $spacing-md;
}
.settings-section-toolbar button {
  background: $theme-palette-bg-secondary;
  color: $theme-palette-text-primary;
  border: none;
  border-radius: $border-radius;
  padding: 4px 12px;
  font-size: $theme-font-size-sm;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}
.settings-section-toolbar button:hover {
  background: $theme-palette-accent;
  color: $theme-palette-text-inverse;
}
.backup-action-button {
  background: $theme-palette-bg-secondary;
  color: $theme-palette-text-primary;
  border: none;
  border-radius: $border-radius;
  padding: 4px 12px;
  font-size: $theme-font-size-sm;
  font-weight: 500;
  cursor: pointer;
  transition:
    background 0.2s,
    color 0.2s;
}
.backup-action-button:hover {
  background: $theme-palette-accent;
  color: $theme-palette-text-inverse;
}
.settings-header {
  margin-bottom: $spacing-xl;
}
@media (prefers-color-scheme: dark) {
  .settings-title {
    color: $theme-palette-text-primary;
  }
  .settings-desc,
  .settings-section-description p,
  .settings-card-desc,
  .settings-section-card p,
  .settings-metric-time,
  .settings-metric-stats td,
  .settings-events-table td,
  .settings-events-table th {
    color: $theme-palette-text-secondary;
  }
  .settings-section-header h2,
  .settings-metric-card h4,
  .settings-metrics-section h3 {
    color: $theme-palette-text-primary;
  }
}
.settings-desc {
  font-size: $theme-font-size-base;
  color: $theme-palette-text-muted;
  margin-top: $spacing-xs;
}

.settings-content.redesigned {
  display: flex;
  flex-direction: column;
  gap: $spacing-2xl;
  background: $theme-palette-bg-nav;
  border-radius: $border-radius-lg;
  padding: $spacing-2xl;
  min-height: 0;
  max-height: 100vh;
  overflow-y: auto;
}
.settings-cards-row {
  display: flex;
  gap: $spacing-2xl;
  margin-bottom: $spacing-2xl;
}
.settings-card {
  background: $theme-palette-bg-primary;
  border-radius: $border-radius-lg;
  box-shadow: 0 2px 16px 0 $theme-palette-border;
  padding: $spacing-xl $spacing-2xl;
  color: $theme-palette-text-primary;
  border: 1px solid $theme-palette-border;
  flex: 1;
  min-width: 320px;
  max-width: 600px;
  transition: box-shadow 0.2s;
}
.settings-card:hover {
  box-shadow: 0 4px 24px 0 $theme-palette-border;
}
.mock-card {
  background: $theme-palette-bg-nav;
  border: 1px dashed $theme-palette-border;
  color: $theme-palette-text-muted;
  margin-top: $spacing-md;
  padding: $spacing-lg;
  border-radius: $border-radius;
  text-align: center;
}
.mock-badge {
  margin-left: 0;
  margin-bottom: 0.5rem;
  background: $theme-palette-bg-secondary;
  color: $theme-palette-text-secondary;
  font-size: $theme-font-size-xs;
  padding: 2px 8px;
  border-radius: 8px;
  text-transform: uppercase;
  font-weight: 700;
  display: inline-block;
}
.mock-loading {
  color: $theme-palette-text-muted;
  font-size: $theme-font-size-base;
  padding: 1.5rem 0;
}
.settings-metrics-section.settings-card {
  margin-top: 0;
  padding: $spacing-xl $spacing-2xl;
}
.settings-metrics-grid.redesigned-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: $spacing-xl;
  margin-bottom: $spacing-xl;
}
.settings-metric-card {
  background: $theme-palette-bg-nav;
  border-radius: $border-radius;
  box-shadow: $shadow-xs;
  padding: $spacing-lg;
  border: 1px solid $theme-palette-border;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}
.settings-metric-card h4 {
  font-size: $theme-font-size-lg;
  font-weight: 600;
  margin-bottom: $spacing-xs;
}
.settings-metric-time {
  font-size: $theme-font-size-xs;
  color: $theme-palette-text-muted;
  margin-bottom: $spacing-xs;
}
.settings-metric-chart {
  background: $theme-palette-bg-primary;
  border-radius: $border-radius;
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: $theme-palette-text-muted;
  font-size: $theme-font-size-sm;
  margin-bottom: $spacing-xs;
}
.settings-metric-filters {
  display: flex;
  gap: $spacing-xs;
  margin-bottom: $spacing-xs;
}
.settings-metric-filters button {
  background: $theme-palette-bg-secondary;
  color: $theme-palette-text-primary;
  border: none;
  border-radius: $border-radius;
  padding: 2px 10px;
  font-size: $theme-font-size-xs;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}
.settings-metric-filters button:hover {
  background: $theme-palette-accent;
  color: $theme-palette-text-inverse;
}
.settings-metric-stats.compact {
  width: 100%;
  font-size: $theme-font-size-xs;
  border-collapse: collapse;
  margin-top: $spacing-xs;
}
.settings-metric-stats.compact td {
  padding: 2px 8px;
  color: $theme-palette-text-secondary;
}
.settings-metric-stats.compact tr {
  border-bottom: 1px solid $theme-palette-border;
}
.settings-events-table.compact {
  width: 100%;
  font-size: $theme-font-size-xs;
  border-collapse: collapse;
  margin-top: $spacing-md;
  background: $theme-palette-bg-nav;
  border-radius: $border-radius;
  overflow: hidden;
}
.settings-events-table.compact th,
.settings-events-table.compact td {
  padding: 6px 10px;
  color: $theme-palette-text-secondary;
  border-bottom: 1px solid $theme-palette-border;
}
.settings-events-table.compact th {
  background: $theme-palette-bg-primary;
  font-weight: 600;
}
.settings-events-actions {
  display: flex;
  gap: $spacing-md;
  margin-top: $spacing-md;
  justify-content: flex-end;
}
</style>
