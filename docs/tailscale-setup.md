# Tailscale Integration

This application includes Tailscale/tsnet integration for secure networking.

## Quick Setup

1. **Get a Tailscale auth key:**
   - Go to https://login.tailscale.com/admin/settings/keys
   - Click "Generate auth key"
   - Check "Reusable" and set an expiration period
   - Copy the generated key

2. **Configure environment variables:**
   
   Create or update your `.env` file:
   ```bash
   # Required
   TAILSCALE_AUTH_KEY=tskey-auth-xxxxxxxxxxxxxxxxxxxxx
   TAILSCALE_HOSTNAME=autobutler-node
   
   # Optional (defaults shown)
   TAILSCALE_CONTROL_URL=https://controlplane.tailscale.com
   TAILSCALE_STATE_DIR=/var/lib/tailscale
   ```

3. **Start the server:**
   ```bash
   make serve/backend
   ```

4. **Verify connection:**
   - Navigate to `/networking` in the UI
   - You should see your node status and Tailscale network info
   - Check your Tailscale admin panel to see the new node

## Configuration Options

| Variable                | Required | Default                              | Description                                    |
| ----------------------- | -------- | ------------------------------------ | ---------------------------------------------- |
| `TAILSCALE_AUTH_KEY`    | Yes      | -                                    | Pre-auth key from Tailscale admin              |
| `TAILSCALE_HOSTNAME`    | Yes      | hostname                             | Name for this node in your tailnet             |
| `TAILSCALE_CONTROL_URL` | No       | `https://controlplane.tailscale.com` | Control server URL (use default for Tailscale) |
| `TAILSCALE_STATE_DIR`   | No       | `/var/lib/tailscale`                 | Directory for Tailscale state                  |

## Using with Headscale

If you want to use a self-hosted Headscale server instead of Tailscale:

```bash
TAILSCALE_CONTROL_URL=https://your-headscale-server.com
TAILSCALE_AUTH_KEY=your-headscale-preauth-key
TAILSCALE_HOSTNAME=autobutler-node
```

## Troubleshooting

**Node not appearing in admin panel:**
- Check that `TAILSCALE_AUTH_KEY` is set correctly
- Verify the auth key hasn't expired
- Check backend logs for connection errors

**"Not configured" message in UI:**
- Ensure `TAILSCALE_AUTH_KEY` is set in your environment
- Restart the backend after changing environment variables
- Check the `.env` file is in the project root

**Connection issues:**
- Verify network connectivity
- Check firewall settings
- Ensure UDP ports are not blocked
