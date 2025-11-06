# Deploying Autobutler to Azure App Service

This guide explains how to deploy Autobutler as a containerized Azure App Service with persistent storage.

## Prerequisites

- Azure CLI installed and logged in (`az login`)
- Docker installed (for building the container image)
- An Azure subscription with appropriate permissions

## Quick Start

### 1. Deploy Azure Container Registry

First, deploy the ACR with the static name `autobutleracr`:

```bash
# Create a resource group
az group create --name autobutler-rg --location eastus

# Deploy the ACR template
az deployment group create \
  --resource-group autobutler-rg \
  --template-file azuredeploy.acr.json \
  --parameters acrSku=Basic
```

### 2. Build and Push the Docker Image

```bash
# Login to the ACR
az acr login --name autobutleracr

# Build the Docker image with version
VERSION=$(git describe --tags --abbrev=0 || echo "dev")
docker build -f cd/azure/Dockerfile -t autobutleracr.azurecr.io/autobutler:latest \
  --build-arg VERSION=$VERSION .

# Push the image
docker push autobutleracr.azurecr.io/autobutler:latest
```

### 3. Deploy the App Service

```bash
# Deploy the app service template
az deployment group create \
  --resource-group autobutler-rg \
  --template-file azuredeploy.json \
  --parameters \
    appName=<your-unique-app-name> \
    storageAccountName=<your-unique-storage-name> \
    dockerImageTag=latest \
    sku=B1
```

**Important:**

- `appName` must be globally unique (used for the URL: `<appName>.azurewebsites.net`)
- `storageAccountName` must be globally unique, 3-24 characters, lowercase letters and numbers only
- The ACR name is fixed as `autobutleracr` and referenced automatically

**Note:** The App Service is automatically configured with a managed identity that has `AcrPull` permissions to the
`autobutleracr` registry. No manual configuration is needed.

## Architecture

The deployment creates:

1. **Azure Container Registry**: `autobutleracr` for storing Docker images
2. **Storage Account**: For persistent data storage
3. **Azure Files Share**: Mounted to `/data` in the container
4. **App Service Plan**: Linux-based hosting plan
5. **App Service**: Container-based web app with managed identity for ACR access

## Configuration

### Environment Variables

The following environment variables are configured by default:

- `PORT=8080` - The port the app listens on
- `DATA_DIR=/data` - The directory for persistent data (mounted from Azure Files)
- `WEBSITES_ENABLE_APP_SERVICE_STORAGE=false` - Use custom storage mount instead

### Volume Mount

Data is persisted in an Azure Files share mounted at `/data`. This includes:

- SQLite databases
- User files
- Any application data

### Pricing

The default configuration uses:

- **App Service Plan**: B1 (Basic) - ~$13/month
- **Storage Account**: Standard LRS - ~$0.05/GB/month

Adjust the `sku` parameter in `azuredeploy.parameters.json` for different performance tiers.

## Updating the Application

### Option 1: Using Azure CLI

```bash
# Build and push new image with tag
VERSION=v1.0.1
docker build -f cd/azure/Dockerfile.azure -t autobutleracr.azurecr.io/autobutler:$VERSION \
  --build-arg VERSION=$VERSION .
docker push autobutleracr.azurecr.io/autobutler:$VERSION

# Update the App Service to use the new tag
az webapp config container set \
  --resource-group autobutler-rg \
  --name <your-app-name> \
  --docker-custom-image-name autobutleracr.azurecr.io/autobutler:$VERSION

# Restart the app
az webapp restart --resource-group autobutler-rg --name <your-app-name>
```

### Option 2: Enable Continuous Deployment

The ARM template sets `DOCKER_ENABLE_CI=true`, which enables webhook-based continuous deployment.
When you push a new image with the same tag to your registry, Azure will automatically pull and deploy it.

## Monitoring

### View Logs

```bash
# Stream logs
az webapp log tail --resource-group autobutler-rg --name <your-app-name>

# Download logs
az webapp log download --resource-group autobutler-rg --name <your-app-name>
```

### Access the Application

After deployment, your app will be available at:

```text
https://<your-app-name>.azurewebsites.net
```

## Troubleshooting

### Container won't start

1. Check the logs: `az webapp log tail --resource-group autobutler-rg --name <your-app-name>`
2. Verify the container image exists and is accessible
3. Ensure the `PORT` environment variable matches what the app listens on (8080)

### Storage mount issues

1. Verify the storage account key is correct
2. Check that the file share exists: `az storage share list --account-name <storage-account-name>`
3. Restart the app service after making storage changes

### Performance issues

Consider upgrading to a higher SKU:

- S1 (Standard) for production workloads
- P1V2 (Premium V2) for high-traffic applications

## Cleanup

To remove all resources:

```bash
az group delete --name autobutler-rg --yes --no-wait
```

## Custom Domain and SSL

To add a custom domain:

```bash
# Add custom domain
az webapp config hostname add \
  --resource-group autobutler-rg \
  --webapp-name <your-app-name> \
  --hostname <your-domain.com>

# Enable SSL (Azure manages the certificate)
az webapp config ssl bind \
  --resource-group autobutler-rg \
  --name <your-app-name> \
  --certificate-thumbprint auto \
  --ssl-type SNI
```
