export interface NetworkNode {
  name: string
  local_ip: string
  ipv6: string
  tailnet_ip?: string
  status: 'online' | 'offline' | 'degraded'
  uptime: number
  throughput_down: number
  throughput_up: number
  latency: number
  configured?: boolean
}

export interface NetworkMetrics {
  active_clients: number
  encrypted_sessions: number
  total_sessions: number
  blocked_requests: number
  last_device_name: string
  last_device_time: string
  throughput_down: number
  throughput_up: number
  latency: number
  uptime: number
}

export interface DiagnosticCheck {
  name: string
  description: string
  status: 'ok' | 'error' | 'warning'
  status_text: string
}

export interface PrivacyFeatures {
  advertise_local: boolean
  remote_tunnel: boolean
  usage_analytics: boolean
}

export interface ConnectionInfo {
  hostname: string
  local_ip: string
  ipv6: string
  tailnet_ip: string
  port: number
  https_url: string
  https_ip_url: string
  tailnet_url: string
  instructions: string[]
}

export const fetchNodeStatus = async (): Promise<NetworkNode> => {
  const res = await fetch('/api/v1/networking/status')
  if (!res.ok) throw new Error('Failed to fetch node status')
  return res.json()
}

export const fetchMetrics = async (): Promise<NetworkMetrics> => {
  const res = await fetch('/api/v1/networking/metrics')
  if (!res.ok) throw new Error('Failed to fetch metrics')
  return res.json()
}

export const fetchDiagnostics = async (): Promise<{ checks: DiagnosticCheck[] }> => {
  const res = await fetch('/api/v1/networking/diagnostics')
  if (!res.ok) throw new Error('Failed to fetch diagnostics')
  return res.json()
}

export const fetchFeatures = async (): Promise<PrivacyFeatures> => {
  const res = await fetch('/api/v1/networking/features')
  if (!res.ok) throw new Error('Failed to fetch features')
  return res.json()
}

export const updateFeatures = async (features: Partial<PrivacyFeatures>): Promise<PrivacyFeatures> => {
  const res = await fetch('/api/v1/networking/features', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(features),
  })
  if (!res.ok) throw new Error('Failed to update features')
  return res.json()
}

export const fetchConnectionInfo = async (): Promise<ConnectionInfo> => {
  const res = await fetch('/api/v1/networking/connection-info')
  if (!res.ok) throw new Error('Failed to fetch connection info')
  return res.json()
}

export interface ConfigurationData {
  headscale_url: string
  auth_key: string
  hostname: string
  state_dir?: string
  webui_port?: number
  environment?: string
  advertise_local?: boolean
  remote_tunnel?: boolean
  usage_analytics?: boolean
  configured?: boolean
}

export const fetchConfiguration = async (): Promise<ConfigurationData> => {
  const res = await fetch('/api/v1/networking/configuration')
  if (!res.ok) throw new Error('Failed to fetch configuration')
  return res.json()
}

export const saveConfiguration = async (config: ConfigurationData): Promise<{ message: string; restart_required: boolean }> => {
  const res = await fetch('/api/v1/networking/configuration', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(config),
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.details || 'Failed to save configuration')
  }
  return res.json()
}

export interface AutoSetupResult {
  headscale_url: string
  auth_key: string
  network_name: string
  node_hostname: string
  instructions: string
}

export const autoSetup = async (): Promise<AutoSetupResult> => {
  const res = await fetch('/api/v1/networking/auto-setup', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  })
  if (!res.ok) {
    const error = await res.json()
    throw new Error(error.error || 'Failed to setup network')
  }
  return res.json()
}
