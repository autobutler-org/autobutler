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
          <button aria-label="Refresh">
            <RefreshIcon />
          </button>
        </div>
      </div>
    </template>
    <template #main>
      <div class="settings-content redesigned">
        <div class="settings-cards-row">
          <section id="storage" class="settings-section settings-card">
            <div class="settings-section-header">
              <StorageDevicesIcon />
              <h2>Storage Devices</h2>
            </div>
            <div class="settings-section-description">
              <p>Manage which storage devices are enabled for Autobutler file storage.</p>
            </div>
            <div class="settings-section-card mock-card">
              <span class="mock-badge">mock</span>
              <p class="mock-loading">Loading devices...</p>
            </div>
          </section>
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

<script setup lang="ts">
import LibraryLayout from '@/components/common/LibraryLayout.vue'
import LibrarySidebar from '@/components/common/LibrarySidebar.vue'
import OpenTelemetryIcon from '@/components/icons/OpenTelemetryIcon.vue'
import RefreshIcon from '@/components/icons/RefreshIcon.vue'
import SaveIcon from '@/components/icons/SaveIcon.vue'
import SearchIcon from '@/components/icons/SearchIcon.vue'
import StorageDevicesIcon from '@/components/icons/StorageDevicesIcon.vue'

const sidebarSections = [
  {
    title: 'Sections',
    items: [
      { label: 'General', active: true },
      { label: 'Users & Access' },
      { label: 'Storage' },
      { label: 'Networking' },
      { label: 'Security' },
      { label: 'OpenTelemetry' },
      { label: 'Advanced' },
    ],
  },
]
</script>

<style lang="scss" scoped>
.settings-sidebar-thanks {
  margin-top: $spacing-2xl;
  padding: $spacing-md;
  border-top: 1px solid $color-gray-800;
}
.settings-thanks-link {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
  color: $color-primary-600;
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
  font-size: $font-size-xs;
  font-weight: 700;
  text-transform: uppercase;
  color: $color-gray-400;
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
  color: $color-gray-300;
  text-decoration: none;
  transition: all 0.2s ease;
  font-size: $font-size-sm;
  cursor: pointer;
}
.library-sidebar-item:hover {
  background: $color-gray-800;
  color: white;
}
.library-sidebar-item.active {
  background: $color-primary-600;
  color: white;
}
.library-sidebar-count {
  margin-left: auto;
  font-size: $font-size-xs;
  color: $color-gray-500;
}
.library-sidebar-item.active .library-sidebar-count {
  color: $color-primary-100;
}
.settings-header {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-lg 0 $spacing-md 0;
  border-bottom: 1px solid $color-gray-800;
  margin-bottom: $spacing-xl;
}
.settings-title {
  font-size: $font-size-3xl;
  font-weight: 700;
  margin: 0;
  color: $color-gray-100;
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
  font-size: $font-size-xl;
  font-weight: 600;
  margin: 0;
  color: $color-gray-100;
}
.settings-section-description {
  margin-bottom: $spacing-md;
  color: $color-gray-400;
}
.settings-section-card {
  background: $color-gray-900;
  border-radius: $border-radius;
  box-shadow: $shadow-sm;
  padding: $spacing-xl;
  color: white;
  border: 1px solid $color-gray-800;
  margin-bottom: $spacing-md;
}
.settings-section-toolbar {
  display: flex;
  gap: $spacing-xs;
  margin-bottom: $spacing-md;
}
.settings-section-toolbar button {
  background: $color-gray-800;
  color: $color-gray-100;
  border: none;
  border-radius: $border-radius;
  padding: 4px 12px;
  font-size: $font-size-sm;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}
.settings-section-toolbar button:hover {
  background: $color-primary-600;
  color: white;
}
.settings-header {
  margin-bottom: $spacing-xl;
}
@media (prefers-color-scheme: dark) {
  .settings-title {
    color: white;
  }
  .settings-desc,
  .settings-section-description p,
  .settings-card-desc,
  .settings-section-card p,
  .settings-metric-time,
  .settings-metric-stats td,
  .settings-events-table td,
  .settings-events-table th {
    color: $color-gray-200;
  }
  .settings-section-header h2,
  .settings-metric-card h4,
  .settings-metrics-section h3 {
    color: white;
  }
}
.settings-desc {
  font-size: $font-size-base;
  color: $color-gray-500;
  margin-top: $spacing-xs;
}

.settings-content.redesigned {
  display: flex;
  flex-direction: column;
  gap: $spacing-2xl;
  background: $color-gray-950;
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
  background: $color-gray-900;
  border-radius: $border-radius-lg;
  box-shadow: 0 2px 16px 0 rgba(0, 0, 0, 0.1);
  padding: $spacing-xl $spacing-2xl;
  color: white;
  border: 1px solid $color-gray-800;
  flex: 1;
  min-width: 320px;
  max-width: 600px;
  transition: box-shadow 0.2s;
}
.settings-card:hover {
  box-shadow: 0 4px 24px 0 rgba(0, 0, 0, 0.18);
}
.mock-card {
  background: $color-gray-950;
  border: 1px dashed $color-gray-700;
  color: $color-gray-400;
  margin-top: $spacing-md;
  padding: $spacing-lg;
  border-radius: $border-radius;
  text-align: center;
}
.mock-badge {
  margin-left: 0;
  margin-bottom: 0.5rem;
  background: $color-gray-300;
  color: $color-gray-700;
  font-size: $font-size-xs;
  padding: 2px 8px;
  border-radius: 8px;
  text-transform: uppercase;
  font-weight: 700;
  display: inline-block;
}
.mock-loading {
  color: $color-gray-500;
  font-size: $font-size-base;
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
  background: $color-gray-950;
  border-radius: $border-radius;
  box-shadow: $shadow-xs;
  padding: $spacing-lg;
  border: 1px solid $color-gray-800;
  display: flex;
  flex-direction: column;
  gap: $spacing-md;
}
.settings-metric-card h4 {
  font-size: $font-size-lg;
  font-weight: 600;
  margin-bottom: $spacing-xs;
}
.settings-metric-time {
  font-size: $font-size-xs;
  color: $color-gray-400;
  margin-bottom: $spacing-xs;
}
.settings-metric-chart {
  background: $color-gray-900;
  border-radius: $border-radius;
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: $color-gray-500;
  font-size: $font-size-sm;
  margin-bottom: $spacing-xs;
}
.settings-metric-filters {
  display: flex;
  gap: $spacing-xs;
  margin-bottom: $spacing-xs;
}
.settings-metric-filters button {
  background: $color-gray-800;
  color: $color-gray-100;
  border: none;
  border-radius: $border-radius;
  padding: 2px 10px;
  font-size: $font-size-xs;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}
.settings-metric-filters button:hover {
  background: $color-primary-600;
  color: white;
}
.settings-metric-stats.compact {
  width: 100%;
  font-size: $font-size-xs;
  border-collapse: collapse;
  margin-top: $spacing-xs;
}
.settings-metric-stats.compact td {
  padding: 2px 8px;
  color: $color-gray-300;
}
.settings-metric-stats.compact tr {
  border-bottom: 1px solid $color-gray-800;
}
.settings-events-table.compact {
  width: 100%;
  font-size: $font-size-xs;
  border-collapse: collapse;
  margin-top: $spacing-md;
  background: $color-gray-950;
  border-radius: $border-radius;
  overflow: hidden;
}
.settings-events-table.compact th,
.settings-events-table.compact td {
  padding: 6px 10px;
  color: $color-gray-300;
  border-bottom: 1px solid $color-gray-800;
}
.settings-events-table.compact th {
  background: $color-gray-900;
  font-weight: 600;
}
.settings-events-actions {
  display: flex;
  gap: $spacing-md;
  margin-top: $spacing-md;
  justify-content: flex-end;
}
</style>
