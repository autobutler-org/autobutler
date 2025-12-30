<script setup lang="ts">
import LibraryLayout from '@/components/common/LibraryLayout.vue'
import LibrarySidebar from '@/components/common/LibrarySidebar.vue'
import {
  fetchDiagnostics,
  fetchFeatures,
  fetchMetrics,
  fetchNodeStatus,
  updateFeatures,
  type DiagnosticCheck,
} from '@/services/networkingService'
import { computed, onMounted, ref } from 'vue'

interface NetworkNode {
  name: string
  localIP: string
  ipv6: string
  tailnetIP?: string
  uptime: number
  throughputDown: number
  throughputUp: number
  latency: number
  status: 'online' | 'offline' | 'degraded'
}

interface PrivacyFeatures {
  advertiseLocal: boolean
  remoteTunnel: boolean
  usageAnalytics: boolean
}

interface NetworkMetrics {
  activeClients: number
  encryptedSessions: number
  totalSessions: number
  lastDeviceName: string
  lastDeviceTime: string
  blockedRequests: number
  blockedTimeframe: string
}

const node = ref<NetworkNode>({
  name: 'orbit-pi.local',
  localIP: '192.168.1.42',
  ipv6: 'fd12:3456:789a::42',
  uptime: 554400000,
  throughputDown: 48,
  throughputUp: 9,
  latency: 6.4,
  status: 'online',
})

const environment = ref('Home')
const loading = ref(true)
const error = ref<string | null>(null)
const isConfigured = ref(true)

const features = ref<PrivacyFeatures>({
  advertiseLocal: true,
  remoteTunnel: true,
  usageAnalytics: true,
})

const metrics = ref<NetworkMetrics>({
  activeClients: 4,
  encryptedSessions: 4,
  totalSessions: 4,
  lastDeviceName: 'MacBook-Pro',
  lastDeviceTime: '14 min ago',
  blockedRequests: 23,
  blockedTimeframe: '24 hr',
})

const diagnostics = ref<DiagnosticCheck[]>([
  {
    name: 'Internet reachability',
    description: 'Ping to 1.1.1.1 and 8.8.8.8 looks healthy.',
    status: 'ok',
    status_text: 'OK',
  },
  {
    name: 'Gateway & DNS',
    description: 'Using 192.168.1.1 as router and DNS resolver.',
    status: 'ok',
    status_text: 'OK',
  },
  {
    name: 'Port exposure',
    description: 'Web UI bound to 8443 - not exposed via UPnP.',
    status: 'warning',
    status_text: 'Locked down',
  },
  {
    name: 'TLS certificates',
    description: 'Local certificate valid - auto-renew in 34 days.',
    status: 'ok',
    status_text: 'Valid',
  },
])

const loadData = async () => {
  try {
    loading.value = true
    error.value = null

    const [statusData, metricsData, diagnosticsData, featuresData] =
      await Promise.all([
        fetchNodeStatus(),
        fetchMetrics(),
        fetchDiagnostics(),
        fetchFeatures(),
      ])

    node.value = {
      name: statusData.name,
      localIP: statusData.local_ip,
      ipv6: statusData.ipv6,
      tailnetIP: statusData.tailnet_ip,
      uptime: statusData.uptime * 1000,
      throughputDown: statusData.throughput_down,
      throughputUp: statusData.throughput_up,
      latency: statusData.latency,
      status: statusData.status,
    }

    isConfigured.value = statusData.configured !== false

    metrics.value = {
      activeClients: metricsData.active_clients,
      encryptedSessions: metricsData.encrypted_sessions,
      totalSessions: metricsData.total_sessions,
      lastDeviceName: metricsData.last_device_name,
      lastDeviceTime: metricsData.last_device_time,
      blockedRequests: metricsData.blocked_requests,
      blockedTimeframe: '24 hr',
    }

    diagnostics.value = diagnosticsData.checks

    features.value = {
      advertiseLocal: featuresData.advertise_local,
      remoteTunnel: featuresData.remote_tunnel,
      usageAnalytics: featuresData.usage_analytics,
    }
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : 'Failed to load networking data'
    console.error('Error loading networking data:', err)
  } finally {
    loading.value = false
  }
}

const handleFeatureToggle = async (featureName: keyof PrivacyFeatures) => {
  try {
    const apiFeatureName =
      featureName === 'advertiseLocal'
        ? 'advertise_local'
        : featureName === 'remoteTunnel'
          ? 'remote_tunnel'
          : 'usage_analytics'

    await updateFeatures({
      [apiFeatureName]: features.value[featureName],
    })
  } catch (err) {
    console.error('Error updating feature:', err)
    // Revert the toggle on error
    features.value[featureName] = !features.value[featureName]
  }
}

onMounted(() => {
  loadData()
  const interval = setInterval(loadData, 10000)
  return () => clearInterval(interval)
})

const uptimeFormatted = computed(() => {
  const ms = node.value.uptime
  const days = Math.floor(ms / (1000 * 60 * 60 * 24))
  const hours = Math.floor((ms % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  return `${days} days ${hours} hr`
})

const statusColor = computed(() => {
  switch (node.value.status) {
    case 'online':
      return '#10b981'
    case 'degraded':
      return '#f59e0b'
    case 'offline':
      return '#ef4444'
  }
})

const sidebarSections = [
  {
    title: 'Sections',
    items: [
      { label: 'General', to: '/settings' },
      { label: 'Users & Access' },
      { label: 'Storage' },
      { label: 'Networking', active: true },
      { label: 'Security' },
      { label: 'OpenTelemetry' },
      { label: 'Advanced' },
    ],
  },
]
</script>

<template>
  <LibraryLayout>
    <template #sidebar>
      <LibrarySidebar :sections="sidebarSections" />
    </template>
    <template #title>
      <h2 class="library-title">Networking</h2>
    </template>
    <template #main>
      <div class="networking-node">
        <div v-if="loading" class="loading-state">
          <p>Loading networking data...</p>
        </div>
        <div v-else-if="error" class="error-state">
          <p>{{ error }}</p>
          <button @click="loadData" class="btn-secondary">Retry</button>
        </div>
        <div v-else-if="!isConfigured" class="setup-container">
          <div class="setup-welcome">
            <div class="welcome-icon">
              <svg width="64" height="64" viewBox="0 0 24 24" fill="none">
                <path
                  d="M12 2L4 6V12C4 16.5 7.5 20.5 12 22C16.5 20.5 20 16.5 20 12V6L12 2Z"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
                <circle
                  cx="12"
                  cy="12"
                  r="3"
                  stroke="currentColor"
                  stroke-width="2"
                />
              </svg>
            </div>
            <h2>Tailscale Not Configured</h2>
            <p class="welcome-description">
              To connect this node to your Tailscale network, configure the
              following environment variables and restart the server:
            </p>

            <div class="env-vars-card">
              <h3>Required Environment Variables</h3>
              <pre><code>TAILSCALE_AUTH_KEY=tskey-auth-xxxxx
TAILSCALE_HOSTNAME=autobutler-node</code></pre>

              <h4>Optional</h4>
              <pre><code>TAILSCALE_CONTROL_URL=https://controlplane.tailscale.com
TAILSCALE_STATE_DIR=/var/lib/tailscale</code></pre>

              <div class="help-text">
                <p><strong>Get your auth key:</strong></p>
                <ol>
                  <li>
                    Go to
                    <a
                      href="https://login.tailscale.com/admin/settings/keys"
                      target="_blank"
                      >Tailscale Admin → Settings → Keys</a
                    >
                  </li>
                  <li>Click "Generate auth key"</li>
                  <li>Check "Reusable" and set expiration</li>
                  <li>Copy the key and add to your .env file</li>
                  <li>Restart the server</li>
                </ol>
              </div>
            </div>
          </div>
        </div>
        <div v-else>
          <div class="header">
            <div class="header-left">
              <div class="shield-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                  <path
                    d="M12 2L4 6V12C4 16.5 7.5 20.5 12 22C16.5 20.5 20 16.5 20 12V6L12 2Z"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                  <path
                    d="M9 12L11 14L15 10"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </div>
              <div class="header-text">
                <h1>Home node · Networking</h1>
                <p>Private, encrypted gateway for your NAS</p>
              </div>
            </div>
            <div class="header-right">
              <span class="badge">Node: {{ node.name.split('.')[0] }}</span>
              <span class="badge">Environment: {{ environment }}</span>
            </div>
          </div>

          <div class="grid">
            <div class="card connection-status">
              <div class="card-header">
                <div>
                  <h2>Connection status</h2>
                  <p class="card-subtitle">
                    Current state of this node on your home network.
                  </p>
                </div>
                <div
                  class="status-badge"
                  :style="{ '--status-color': statusColor }"
                >
                  <span class="status-dot"></span>
                  Healthy · {{ node.status }}
                </div>
              </div>

              <div class="node-info">
                <h3>{{ node.name }}</h3>
                <p class="ip-addresses">
                  {{ node.localIP }} · IPv6: {{ node.ipv6 }}
                </p>
              </div>

              <div class="progress-bar">
                <div class="progress-fill"></div>
              </div>

              <div class="metrics">
                <div class="metric">
                  <span class="metric-label">Uptime</span>
                  <span class="metric-value">{{ uptimeFormatted }}</span>
                </div>
                <div class="metric">
                  <span class="metric-label">Current throughput</span>
                  <span class="metric-value"
                    >{{ node.throughputDown }} Mbps ↓ ·
                    {{ node.throughputUp }} Mbps ↑</span
                  >
                </div>
                <div class="metric">
                  <span class="metric-label">Latency to gateway</span>
                  <span class="metric-value">{{ node.latency }} ms</span>
                </div>
              </div>
            </div>

            <div class="right-column">
              <div class="card privacy-features">
                <div class="card-header">
                  <div>
                    <h2>Privacy & features</h2>
                    <p class="card-subtitle">
                      Keep the node quiet on the network until you need it.
                    </p>
                  </div>
                </div>

                <div class="feature-toggles">
                  <div class="feature-item">
                    <div class="feature-info">
                      <h4>Advertise on local network</h4>
                      <p>
                        Broadcasts service via mDNS / Bonjour so devices can
                        discover it.
                      </p>
                    </div>
                    <label class="toggle">
                      <input
                        type="checkbox"
                        v-model="features.advertiseLocal"
                        @change="handleFeatureToggle('advertiseLocal')"
                      />
                      <span class="toggle-slider"></span>
                    </label>
                  </div>

                  <div class="feature-item">
                    <div class="feature-info">
                      <h4>Remote access tunnel</h4>
                      <p>
                        Create an end-to-end encrypted tunnel for access outside
                        your home.
                      </p>
                    </div>
                    <label class="toggle">
                      <input
                        type="checkbox"
                        v-model="features.remoteTunnel"
                        @change="handleFeatureToggle('remoteTunnel')"
                      />
                      <span class="toggle-slider"></span>
                    </label>
                  </div>

                  <div class="feature-item">
                    <div class="feature-info">
                      <h4>Usage analytics</h4>
                      <p>
                        Collect anonymous metrics locally only. Nothing leaves
                        your network.
                      </p>
                    </div>
                    <label class="toggle">
                      <input
                        type="checkbox"
                        v-model="features.usageAnalytics"
                        @change="handleFeatureToggle('usageAnalytics')"
                      />
                      <span class="toggle-slider"></span>
                    </label>
                  </div>
                </div>

                <div class="metrics-grid">
                  <div class="metric-item">
                    <span class="metric-label">Active clients</span>
                    <span class="metric-value"
                      >{{ metrics.activeClients }} devices</span
                    >
                  </div>
                  <div class="metric-item">
                    <span class="metric-label">Encrypted sessions</span>
                    <span class="metric-value"
                      >{{ metrics.encryptedSessions }} /
                      {{ metrics.totalSessions }} TLS</span
                    >
                  </div>
                  <div class="metric-item">
                    <span class="metric-label">Last new device</span>
                    <span class="metric-value"
                      >{{ metrics.lastDeviceName }} ·
                      {{ metrics.lastDeviceTime }}</span
                    >
                  </div>
                  <div class="metric-item">
                    <span class="metric-label">Blocked requests</span>
                    <span class="metric-value"
                      >{{ metrics.blockedRequests }} in last
                      {{ metrics.blockedTimeframe }}</span
                    >
                  </div>
                </div>
              </div>

              <div class="card diagnostics">
                <div class="card-header">
                  <div>
                    <h2>Diagnostics</h2>
                    <p class="card-subtitle">
                      Quick checks to understand what might be wrong.
                    </p>
                  </div>
                </div>

                <div class="diagnostic-checks">
                  <div
                    v-for="check in diagnostics"
                    :key="check.name"
                    class="diagnostic-item"
                  >
                    <div class="diagnostic-info">
                      <h4>{{ check.name }}</h4>
                      <p>{{ check.description }}</p>
                    </div>
                    <span class="diagnostic-status" :data-status="check.status">
                      {{ check.status_text }}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div class="card how-to-connect">
              <div class="card-header">
                <div>
                  <h2>How to connect</h2>
                  <p class="card-subtitle">
                    Simple instructions you can share with anyone on your
                    network.
                  </p>
                </div>
              </div>

              <div class="connection-steps">
                <div class="step">
                  <span class="step-number">1</span>
                  <div class="step-content">
                    <h4>From your laptop or phone</h4>
                    <p>
                      Make sure you're on the same Wi-Fi or Ethernet network as
                      this node.
                    </p>
                  </div>
                </div>

                <div class="step">
                  <span class="step-number">2</span>
                  <div class="step-content">
                    <h4>Open this address</h4>
                    <p>Type the following URL into your browser.</p>
                    <code class="connection-url"
                      >https://{{ node.name }}:8443</code
                    >
                  </div>
                </div>

                <div class="step">
                  <span class="step-number">3</span>
                  <div class="step-content">
                    <h4>Optional: direct IP access</h4>
                    <p>
                      If mDNS/local name resolution fails, use the IP address
                      instead.
                    </p>
                    <code class="connection-url"
                      >https://{{ node.localIP }}:8443</code
                    >
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="footer">
            <p>
              All diagnostics run locally on this node. No traffic metadata is
              sent to third parties.
            </p>
            <div class="footer-actions">
              <button class="btn-secondary">View raw logs</button>
              <button class="btn-secondary">Export config</button>
            </div>
          </div>
        </div>
      </div>
    </template>
  </LibraryLayout>
</template>

<style scoped lang="scss">
.networking-node {
  padding: $spacing-2xl;
  color: $color-gray-200;
  background: $color-gray-950;
  border-radius: $border-radius-lg;
  min-height: 0;
  height: 100%;
  overflow-y: auto;
}

@media (prefers-color-scheme: light) {
  .networking-node {
    color: hsl(220, 15%, 20%);
    background: $color-gray-100;
  }
}

.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  gap: 1rem;
}

.loading-state p {
  font-size: 1rem;
  color: hsl(220, 10%, 60%);
}

.error-state p {
  font-size: 1rem;
  color: hsl(0, 60%, 60%);
}

.config-warning {
  display: flex;
  flex-direction: column;
  gap: 2rem;
  padding: 2rem;
  background: hsl(220, 15%, 12%);
  border: 1px solid hsl(220, 15%, 20%);
  border-radius: 0.75rem;
  margin-bottom: 2rem;
}

.warning-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1rem 1.5rem;
  background: hsl(45, 100%, 15%);
  border: 1px solid hsl(45, 100%, 25%);
  border-radius: 0.5rem;
  color: hsl(45, 100%, 80%);
}

.config-warning svg {
  flex-shrink: 0;
  margin-top: 0.125rem;
}

.warning-header h3 {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.warning-header p {
  font-size: 0.875rem;
  line-height: 1.5;
  margin: 0;
}

.config-form {
  display: flex;
  flex-direction: column;
  gap: 2rem;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.form-section h4 {
  font-size: 1rem;
  font-weight: 600;
  color: $color-gray-200;
  margin-bottom: 0.5rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-group label {
  font-size: 0.875rem;
  font-weight: 500;
  color: $color-gray-300;
}

.form-group input {
  padding: 0.75rem 1rem;
  background: hsl(220, 15%, 16%);
  border: 1px solid hsl(220, 15%, 25%);
  border-radius: 0.5rem;
  color: $color-gray-100;
  font-size: 0.9375rem;
  font-family: inherit;
  transition:
    border-color 0.2s,
    background 0.2s;
}

.form-group input:focus {
  outline: none;
  border-color: hsl(220, 70%, 50%);
  background: hsl(220, 15%, 18%);
}

.form-hint {
  font-size: 0.8125rem;
  color: $color-gray-500;
  line-height: 1.4;
}

.form-actions {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding-top: 1rem;
}

.btn-primary {
  padding: 0.875rem 1.5rem;
  background: hsl(220, 70%, 50%);
  color: white;
  border: none;
  border-radius: 0.5rem;
  font-size: 0.9375rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
  align-self: flex-start;
}

.btn-primary:hover {
  background: hsl(220, 70%, 45%);
}

.btn-primary:disabled {
  background: hsl(220, 15%, 30%);
  cursor: not-allowed;
  opacity: 0.6;
}

.error-message {
  color: hsl(0, 70%, 60%);
  font-size: 0.875rem;
  margin: 0;
}

.success-message {
  color: hsl(120, 50%, 50%);
  font-size: 0.875rem;
  margin: 0;
}

.link-button {
  background: none;
  border: none;
  color: hsl(220, 70%, 60%);
  text-decoration: underline;
  cursor: pointer;
  padding: 0;
  font-size: inherit;
  font-family: inherit;
}

.link-button:hover {
  color: hsl(220, 70%, 50%);
}

.setup-guide {
  padding: 1.5rem;
  background: hsl(220, 15%, 14%);
  border: 1px solid hsl(220, 15%, 22%);
  border-radius: 0.5rem;
}

.setup-guide h4 {
  font-size: 1.125rem;
  font-weight: 600;
  color: $color-gray-100;
  margin-bottom: 1rem;
}

.guide-intro {
  font-size: 0.9375rem;
  color: $color-gray-300;
  line-height: 1.6;
  margin-bottom: 1.5rem;
}

.guide-steps {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.guide-step {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
}

.guide-step .step-number {
  flex-shrink: 0;
  width: 2rem;
  height: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: hsl(220, 70%, 50%);
  color: white;
  border-radius: 50%;
  font-weight: 600;
  font-size: 0.875rem;
}

.guide-step .step-content {
  flex: 1;
}

.guide-step h5 {
  font-size: 0.9375rem;
  font-weight: 600;
  color: $color-gray-200;
  margin-bottom: 0.5rem;
}

.guide-step pre {
  background: hsl(220, 15%, 10%);
  padding: 0.75rem 1rem;
  border-radius: 0.375rem;
  overflow-x: auto;
  margin: 0.5rem 0;
}

.guide-step code {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.8125rem;
  color: hsl(120, 50%, 60%);
  line-height: 1.5;
}

.step-note {
  font-size: 0.8125rem;
  color: $color-gray-500;
  line-height: 1.4;
  margin: 0.5rem 0 0 0;
}

.step-note code {
  background: hsl(220, 15%, 18%);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.75rem;
  color: hsl(220, 70%, 60%);
}

.step-note a {
  color: hsl(220, 70%, 60%);
  text-decoration: underline;
}

.step-note a:hover {
  color: hsl(220, 70%, 50%);
}

.guide-footer {
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid hsl(220, 15%, 22%);
  color: $color-gray-400;
  font-size: 0.875rem;
}

@media (prefers-color-scheme: light) {
  .link-button {
    color: hsl(220, 70%, 45%);
  }

  .link-button:hover {
    color: hsl(220, 70%, 35%);
  }

  .setup-guide {
    background: white;
    border-color: $color-gray-200;
  }

  .setup-guide h4 {
    color: $color-gray-900;
  }

  .guide-intro {
    color: $color-gray-700;
  }

  .guide-step h5 {
    color: $color-gray-800;
  }

  .guide-step pre {
    background: $color-gray-100;
  }

  .guide-step code {
    color: hsl(120, 50%, 35%);
  }

  .step-note {
    color: $color-gray-600;
  }

  .step-note code {
    background: $color-gray-100;
    color: hsl(220, 70%, 45%);
  }

  .step-note a {
    color: hsl(220, 70%, 45%);
  }

  .guide-footer {
    border-top-color: $color-gray-200;
    color: $color-gray-600;
  }
}

@media (prefers-color-scheme: light) {
  .config-warning {
    background: $color-gray-50;
    border-color: $color-gray-200;
  }

  .warning-header {
    background: hsl(45, 100%, 95%);
    border-color: hsl(45, 100%, 75%);
    color: hsl(45, 100%, 30%);
  }

  .form-section h4 {
    color: $color-gray-800;
  }

  .form-group label {
    color: $color-gray-700;
  }

  .form-group input {
    background: white;
    border-color: $color-gray-300;
    color: $color-gray-900;
  }

  .form-group input:focus {
    border-color: hsl(220, 70%, 50%);
    background: white;
  }

  .btn-primary:disabled {
    background: $color-gray-300;
  }
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
  gap: 1rem;
  flex-wrap: wrap;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.shield-icon {
  width: 2.5rem;
  height: 2.5rem;
  background: hsl(150, 60%, 45%);
  border-radius: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.header-text h1 {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 0.25rem;
}

.header-text p {
  font-size: 0.875rem;
  color: hsl(220, 10%, 60%);
}

@media (prefers-color-scheme: light) {
  .header-text p {
    color: hsl(220, 10%, 50%);
  }
}

.header-right {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.badge {
  padding: 0.5rem 1rem;
  background: hsl(220, 20%, 18%);
  border-radius: 0.375rem;
  font-size: 0.875rem;
  color: hsl(220, 15%, 75%);
}

@media (prefers-color-scheme: light) {
  .badge {
    background: hsl(220, 20%, 92%);
    color: hsl(220, 15%, 35%);
  }
}

.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  margin-bottom: 1.5rem;
}

@media (max-width: 1200px) {
  .grid {
    grid-template-columns: 1fr;
  }
}

.right-column {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.card {
  background: hsl(220, 20%, 15%);
  border-radius: 1rem;
  padding: 1.5rem;
  border: 1px solid hsl(220, 20%, 20%);
}

@media (prefers-color-scheme: light) {
  .card {
    background: white;
    border-color: hsl(220, 20%, 88%);
  }
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.5rem;
  gap: 1rem;
}

.card-header h2 {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 0.25rem;
}

.card-subtitle {
  font-size: 0.875rem;
  color: hsl(220, 10%, 55%);
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: hsl(150, 50%, 20%);
  border-radius: 0.5rem;
  font-size: 0.875rem;
  color: hsl(150, 60%, 70%);
  white-space: nowrap;
}

@media (prefers-color-scheme: light) {
  .status-badge {
    background: hsl(150, 50%, 95%);
    color: hsl(150, 60%, 35%);
  }
}

.status-dot {
  width: 0.5rem;
  height: 0.5rem;
  background: var(--status-color, #10b981);
  border-radius: 50%;
}

.node-info h3 {
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 0.25rem;
}

.ip-addresses {
  font-size: 0.875rem;
  color: hsl(220, 10%, 60%);
  font-family: 'SF Mono', Monaco, monospace;
}

.progress-bar {
  height: 0.375rem;
  background: hsl(220, 20%, 20%);
  border-radius: 0.25rem;
  margin: 1.5rem 0;
  overflow: hidden;
}

@media (prefers-color-scheme: light) {
  .progress-bar {
    background: hsl(220, 20%, 90%);
  }
}

.progress-fill {
  height: 100%;
  width: 75%;
  background: linear-gradient(90deg, hsl(150, 60%, 45%), hsl(150, 60%, 55%));
  border-radius: 0.25rem;
}

.metrics {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1.5rem;
}

@media (max-width: 768px) {
  .metrics {
    grid-template-columns: 1fr;
  }
}

.metric {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.metric-label {
  font-size: 0.75rem;
  color: hsl(220, 10%, 55%);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.metric-value {
  font-size: 0.9375rem;
  font-weight: 500;
}

.feature-toggles {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.feature-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
}

.feature-info h4 {
  font-size: 0.9375rem;
  font-weight: 500;
  margin-bottom: 0.25rem;
}

.feature-info p {
  font-size: 0.8125rem;
  color: hsl(220, 10%, 55%);
  line-height: 1.4;
}

.toggle {
  position: relative;
  display: inline-block;
  width: 2.75rem;
  height: 1.5rem;
  flex-shrink: 0;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: hsl(220, 15%, 25%);
  transition: 0.2s;
  border-radius: 1.5rem;
}

@media (prefers-color-scheme: light) {
  .toggle-slider {
    background: hsl(220, 15%, 80%);
  }
}

.toggle-slider::before {
  position: absolute;
  content: '';
  height: 1.125rem;
  width: 1.125rem;
  left: 0.1875rem;
  bottom: 0.1875rem;
  background: white;
  transition: 0.2s;
  border-radius: 50%;
}

.toggle input:checked + .toggle-slider {
  background: hsl(150, 60%, 45%);
}

.toggle input:checked + .toggle-slider::before {
  transform: translateX(1.25rem);
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
  padding: 1rem;
  background: hsl(220, 20%, 12%);
  border-radius: 0.5rem;
}

@media (prefers-color-scheme: light) {
  .metrics-grid {
    background: hsl(220, 20%, 97%);
  }
}

.metric-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.diagnostic-checks {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.diagnostic-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
}

.diagnostic-info h4 {
  font-size: 0.9375rem;
  font-weight: 500;
  margin-bottom: 0.25rem;
}

.diagnostic-info p {
  font-size: 0.8125rem;
  color: hsl(220, 10%, 55%);
  line-height: 1.4;
}

.diagnostic-status {
  padding: 0.375rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.8125rem;
  font-weight: 500;
  white-space: nowrap;

  &[data-status='ok'] {
    background: hsl(150, 50%, 20%);
    color: hsl(150, 60%, 70%);
  }

  &[data-status='warning'] {
    background: hsl(45, 50%, 20%);
    color: hsl(45, 60%, 70%);
  }

  &[data-status='error'] {
    background: hsl(0, 50%, 20%);
    color: hsl(0, 60%, 70%);
  }
}

@media (prefers-color-scheme: light) {
  .diagnostic-status {
    &[data-status='ok'] {
      background: hsl(150, 50%, 95%);
      color: hsl(150, 60%, 35%);
    }

    &[data-status='warning'] {
      background: hsl(45, 50%, 95%);
      color: hsl(45, 60%, 35%);
    }

    &[data-status='error'] {
      background: hsl(0, 50%, 95%);
      color: hsl(0, 60%, 35%);
    }
  }
}

.connection-steps {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.step {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
}

.step-number {
  width: 1.75rem;
  height: 1.75rem;
  background: hsl(220, 20%, 20%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.875rem;
  font-weight: 600;
  flex-shrink: 0;
}

@media (prefers-color-scheme: light) {
  .step-number {
    background: hsl(220, 20%, 90%);
  }
}

.step-content h4 {
  font-size: 0.9375rem;
  font-weight: 500;
  margin-bottom: 0.375rem;
}

.step-content p {
  font-size: 0.875rem;
  color: hsl(220, 10%, 55%);
  margin-bottom: 0.5rem;
}

.connection-url {
  display: inline-block;
  padding: 0.5rem 0.75rem;
  background: hsl(220, 20%, 12%);
  border-radius: 0.375rem;
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.875rem;
  color: hsl(150, 60%, 60%);
}

@media (prefers-color-scheme: light) {
  .connection-url {
    background: hsl(220, 20%, 97%);
    color: hsl(150, 60%, 40%);
  }
}

.how-to-connect {
  grid-column: 1 / -1;
}

.footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 0;
  gap: 1rem;
  flex-wrap: wrap;
}

.footer p {
  font-size: 0.8125rem;
  color: hsl(220, 10%, 55%);
}

.footer-actions {
  display: flex;
  gap: 0.75rem;
}

.btn-secondary {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid hsl(220, 20%, 25%);
  border-radius: 0.375rem;
  font-size: 0.875rem;
  color: hsl(220, 15%, 75%);
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background: hsl(220, 20%, 18%);
    border-color: hsl(220, 20%, 30%);
  }
}

@media (prefers-color-scheme: light) {
  .btn-secondary {
    border-color: hsl(220, 20%, 80%);
    color: hsl(220, 15%, 35%);

    &:hover {
      background: hsl(220, 20%, 95%);
      border-color: hsl(220, 20%, 75%);
    }
  }
}

.env-vars-card {
  background: hsl(220, 15%, 12%);
  border: 1px solid hsl(220, 15%, 20%);
  border-radius: 0.75rem;
  padding: 2rem;
  text-align: left;
  max-width: 700px;
  margin: 0 auto;

  h3 {
    font-size: 1rem;
    font-weight: 600;
    color: $color-gray-200;
    margin-bottom: 1rem;
  }

  h4 {
    font-size: 0.9rem;
    font-weight: 600;
    color: $color-gray-300;
    margin: 1.5rem 0 0.75rem;
  }

  pre {
    background: hsl(220, 15%, 10%);
    padding: 1rem 1.25rem;
    border-radius: 0.5rem;
    overflow-x: auto;
    margin-bottom: 1.5rem;
  }

  code {
    font-family: 'SF Mono', Monaco, monospace;
    font-size: 0.875rem;
    color: hsl(120, 50%, 60%);
    line-height: 1.6;
  }

  @media (prefers-color-scheme: light) {
    background: $color-gray-50;
    border-color: $color-gray-200;

    h3 {
      color: $color-gray-900;
    }

    h4 {
      color: $color-gray-800;
    }

    pre {
      background: $color-gray-100;
    }

    code {
      color: hsl(120, 50%, 35%);
    }
  }
}

.help-text {
  padding-top: 1.5rem;
  border-top: 1px solid hsl(220, 15%, 20%);

  p {
    font-size: 0.9375rem;
    color: $color-gray-300;
    margin-bottom: 0.75rem;
  }

  ol {
    padding-left: 1.5rem;
    color: $color-gray-400;
    line-height: 1.8;
    font-size: 0.9375rem;
  }

  li {
    margin-bottom: 0.5rem;
  }

  a {
    color: hsl(220, 70%, 60%);
    text-decoration: underline;

    &:hover {
      color: hsl(220, 70%, 50%);
    }
  }

  @media (prefers-color-scheme: light) {
    border-top-color: $color-gray-200;

    p {
      color: $color-gray-700;
    }

    ol {
      color: $color-gray-600;
    }

    a {
      color: hsl(220, 70%, 45%);
    }
  }
}
</style>
