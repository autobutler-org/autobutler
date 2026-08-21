# Quark Headscale — Azure ARM Deployment

Deploys a lightweight Ubuntu 24.04 VM on Azure to host:

- **[Headscale](https://github.com/juanfont/headscale)** — self-hosted Tailscale control server
- **Quark Provisioning Service** — auto-generates pre-auth keys for new devices

## Architecture

```
Internet
   │
   ├── :443  ──▶  Nginx  ──▶  Headscale  (127.0.0.1:8080)
   ├── :80   ──▶  Nginx  ──▶  Headscale  (redirect / certbot ACME)
   ├── :8081 ──▶  Quark Provisioning Service
   └── :3478 (UDP) ──▶  Headscale STUN
```

Data flows peer-to-peer over WireGuard — **the VM only handles control plane traffic**, not file data.

## Prerequisites

1. Azure subscription with a resource group
2. DNS A record for your `headscaleDomain` pointed at the VM's public IP  
   _(certbot won't issue TLS until DNS resolves)_
3. SSH keypair (`ssh-keygen -t ed25519`)
4. Azure CLI installed: `az login`

## Deploy

```bash
# Create resource group (if needed)
az group create --name quark-headscale-rg --location eastus

# Deploy
az deployment group create \
  --resource-group quark-headscale-rg \
  --template-file headscale.json \
  --parameters headscale.parameters.json \
  --parameters adminPublicKey="$(cat ~/.ssh/id_ed25519.pub)" \
               headscaleDomain="ts.quark.org"
```

## Post-Deploy: Generate Headscale API Key

Once the VM is up, SSH in and generate the API key for the provisioning service:

```bash
ssh quark@<VM_PUBLIC_IP>

# Generate an API key
sudo headscale apikeys create --expiration 9999d

# Set it in the provisioning service env and start
sudo systemctl edit quark-provisioning
# Add under [Service]:
#   Environment=HEADSCALE_API_KEY=<your-key>

sudo systemctl start quark-provisioning
sudo systemctl status quark-provisioning
```

## Post-Deploy: Create a Tailnet

```bash
# Create the quark namespace/user
sudo headscale users create quark

# Verify Headscale is running
sudo headscale version
sudo headscale nodes list
```

## Parameters

| Parameter         | Default                 | Description                                  |
| ----------------- | ----------------------- | -------------------------------------------- |
| `vmName`          | `quark-headscale`       | VM and resource name prefix                  |
| `adminUsername`   | `quark`            | SSH admin user                               |
| `adminPublicKey`  | _(required)_            | SSH public key content                       |
| `vmSize`          | `Standard_B1s`          | Azure VM SKU (1 vCPU / 1 GB RAM)             |
| `location`        | resource group location | Azure region                                 |
| `headscaleDomain` | _(required)_            | Public domain (e.g. `ts.quark.org`)     |
| `headscaleApiKey` | `""`                    | Set after first boot (see post-deploy steps) |
| `allowedSSHCidr`  | `*`                     | Restrict SSH to your IP for production       |

## Ports

| Port | Protocol | Purpose                             |
| ---- | -------- | ----------------------------------- |
| 22   | TCP      | SSH                                 |
| 80   | TCP      | HTTP / Let's Encrypt ACME challenge |
| 443  | TCP      | Headscale HTTPS (via Nginx)         |
| 3478 | UDP      | Headscale STUN (NAT traversal)      |
| 8081 | TCP      | Quark Provisioning Service          |

## Cost Estimate

`Standard_B1s` in East US: ~**$8–10/month** (or ~$4–5 with a 1-year reserved instance).

## Teardown

```bash
az group delete --name quark-headscale-rg --yes
```
