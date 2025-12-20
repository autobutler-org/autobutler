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
    <template #main>
      <div class="settings-header">
        <h1 class="settings-title">Settings & Metrics</h1>
        <div class="settings-header-actions">
          <button aria-label="Save">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              width="20"
              height="20"
            >
              <path d="M19 21H5a2 2 0 01-2-2V5a2 2 0 012-2h11l5 5v11a2 2 0 01-2 2z"></path>
              <polyline points="17 21 17 13 7 13 7 21"></polyline>
              <polyline points="7 3 7 8 15 8"></polyline>
            </svg>
          </button>
          <button aria-label="Refresh">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              width="20"
              height="20"
            >
              <polyline points="23 4 23 10 17 10"></polyline>
              <path d="M20.49 15a9 9 0 11-2.12-9.36L23 10"></path>
            </svg>
          </button>
        </div>
      </div>
      <div class="settings-content">
        <section id="storage" class="settings-section">
          <div class="settings-section-header">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              width="20"
              height="20"
            >
              <ellipse cx="12" cy="5" rx="9" ry="3"></ellipse>
              <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path>
              <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>
            </svg>
            <h2>Storage Devices</h2>
          </div>
          <div class="settings-section-description">
            <p>Manage which storage devices are enabled for Autobutler file storage.</p>
          </div>
          <div class="settings-section-card">
            <span class="mock-badge">mock</span>
            <p style="text-align: center; padding: 2rem; color: var(--color-gray-500)">
              Loading devices...
            </p>
          </div>
        </section>
        <section id="opentelemetry" class="settings-section">
          <div class="settings-section-header">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              width="20"
              height="20"
            >
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>
            </svg>
            <h2>OpenTelemetry</h2>
          </div>
          <div class="settings-section-toolbar">
            <button>Configuration</button>
            <button>Metrics</button>
            <button>Traces</button>
            <button>Logs</button>
            <button>
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                width="16"
                height="16"
              >
                <circle cx="11" cy="11" r="8"></circle>
                <path d="M21 21l-4.35-4.35"></path>
              </svg>
              Search settings
            </button>
          </div>
          <div class="settings-section-card">
            <span class="mock-badge">mock</span>
            <h3>Collector & Exporter</h3>
            <p>Configure where telemetry is sent and how it's batched</p>
          </div>
        </section>
        <section class="settings-metrics-section">
          <span class="mock-badge">mock</span>
          <h3>OpenTelemetry Metrics</h3>
          <p>Live instrumentation snapshots for the NAS and file browser.</p>
          <div class="settings-metrics-grid">
            <div class="settings-metric-card">
              <h4>CPU Usage (%)</h4>
              <span class="settings-metric-time">Last 15m</span>
              <div class="settings-metric-chart">Line chart placeholder</div>
              <div class="settings-metric-filters">
                <button>System</button>
                <button>Collector</button>
              </div>
              <table class="settings-metric-stats">
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
              <table class="settings-metric-stats">
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
              <table class="settings-metric-stats">
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
              <table class="settings-metric-stats">
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
          <table class="settings-events-table">
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

<style scoped>
.settings-sidebar-thanks {
  margin-top: var(--spacing-2xl);
  padding: var(--spacing-md);
  border-top: 1px solid var(--color-gray-800);
}
.settings-thanks-link {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  color: var(--color-primary-600);
  text-decoration: none;
  font-weight: 600;
  margin-top: var(--spacing-xs);
}
.settings-thanks-link:hover {
  text-decoration: underline;
}
.library-sidebar {
  padding-top: var(--spacing-xl);
}
.library-sidebar-section {
  margin-bottom: var(--spacing-xl);
}
.library-sidebar-title {
  font-size: var(--font-size-xs);
  font-weight: 700;
  text-transform: uppercase;
  color: var(--color-gray-400);
  margin-bottom: var(--spacing-xs);
  letter-spacing: 0.05em;
}
.library-sidebar-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}
.library-sidebar-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-md);
  border-radius: var(--border-radius);
  color: var(--color-gray-300);
  text-decoration: none;
  transition: all 0.2s ease;
  font-size: var(--font-size-sm);
  cursor: pointer;
}
.library-sidebar-item:hover {
  background: var(--color-gray-800);
  color: white;
}
.library-sidebar-item.active {
  background: var(--color-primary-600);
  color: white;
}
.library-sidebar-count {
  margin-left: auto;
  font-size: var(--font-size-xs);
  color: var(--color-gray-500);
}
.library-sidebar-item.active .library-sidebar-count {
  color: var(--color-primary-100);
}
.settings-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg) 0 var(--spacing-md) 0;
  border-bottom: 1px solid var(--color-gray-800);
  margin-bottom: var(--spacing-xl);
}
.settings-title {
  font-size: var(--font-size-3xl);
  font-weight: 700;
  margin: 0;
  color: var(--color-gray-100);
}
.settings-header-actions {
  display: flex;
  gap: var(--spacing-md);
}
.settings-section {
  margin-bottom: var(--spacing-2xl);
}
.settings-section-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  margin-bottom: var(--spacing-xs);
}
.settings-section-header h2 {
  font-size: var(--font-size-xl);
  font-weight: 600;
  margin: 0;
  color: var(--color-gray-100);
}
.settings-section-description {
  margin-bottom: var(--spacing-md);
  color: var(--color-gray-400);
}
.settings-section-card {
  background: var(--color-gray-900);
  border-radius: var(--border-radius);
  box-shadow: var(--shadow-sm);
  padding: var(--spacing-xl);
  color: white;
  border: 1px solid var(--color-gray-800);
  margin-bottom: var(--spacing-md);
}
.settings-section-toolbar {
  display: flex;
  gap: var(--spacing-xs);
  margin-bottom: var(--spacing-md);
}
.settings-section-toolbar button {
  background: var(--color-gray-800);
  color: var(--color-gray-100);
  border: none;
  border-radius: var(--border-radius);
  padding: 4px 12px;
  font-size: var(--font-size-sm);
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}
.settings-section-toolbar button:hover {
  background: var(--color-primary-600);
  color: white;
}
.settings-header {
  margin-bottom: var(--spacing-xl);
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
    color: var(--color-gray-200);
  }
  .settings-section-header h2,
  .settings-metric-card h4,
  .settings-metrics-section h3 {
    color: white;
  }
}
.settings-desc {
  font-size: var(--font-size-base);
  color: var(--color-gray-500);
  margin-top: var(--spacing-xs);
}
.settings-content {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: var(--spacing-xl);
  margin-top: var(--spacing-lg);
}
.settings-card {
  background: var(--color-gray-900);
  border-radius: var(--border-radius);
  box-shadow: var(--shadow-sm);
  padding: var(--spacing-xl);
  color: white;
  border: 1px solid var(--color-gray-800);
}
.settings-card-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  margin-bottom: var(--spacing-xs);
}
.settings-card-desc {
  font-size: var(--font-size-base);
  color: var(--color-gray-300);
}
.mock-badge {
  margin-left: 1rem;
  background: var(--color-gray-300);
  color: var(--color-gray-700);
  font-size: var(--font-size-xs);
  padding: 2px 8px;
  border-radius: 8px;
  text-transform: uppercase;
  font-weight: 700;
}
</style>
