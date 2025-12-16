import { test, expect } from '@playwright/test';

test.describe('Device Management', () => {
    test('settings page has storage section', async ({ page }) => {
        await page.goto('/settings');

        // Check for storage section
        const storageSection = page.locator('#storage');
        await expect(storageSection).toBeVisible();

        // Check section header
        const header = storageSection.locator('.settings-section-header h2');
        await expect(header).toHaveText('Storage Devices');
    });

    test('device manager component loads with clear description', async ({ page }) => {
        await page.goto('/settings');

        // Wait for device manager to load via HTMX
        const deviceManager = page.locator('.device-manager');
        await expect(deviceManager).toBeVisible({ timeout: 5000 });

        // Check for title
        const title = deviceManager.locator('.device-manager-title');
        await expect(title).toHaveText('Storage Device Management');

        // Check for description with technical details
        const description = deviceManager.locator('.device-manager-description');
        await expect(description).toContainText('data/cirrus');
        await expect(description).toContainText('Enabled devices');
    });

    test('displays detected storage devices', async ({ page }) => {
        await page.goto('/settings');

        // Wait for device manager
        const deviceManager = page.locator('.device-manager');
        await expect(deviceManager).toBeVisible({ timeout: 5000 });

        // Check for device list
        const deviceList = deviceManager.locator('.device-manager-list');
        await expect(deviceList).toBeVisible();

        // Should have at least one device (the system drive)
        const devices = deviceManager.locator('.device-manager-item');
        await expect(devices.first()).toBeVisible();
    });

    test('device items show name and details', async ({ page }) => {
        await page.goto('/settings');

        await page.locator('.device-manager').waitFor({ state: 'visible', timeout: 5000 });

        const firstDevice = page.locator('.device-manager-item').first();
        await expect(firstDevice).toBeVisible();

        // Check for device name
        const name = firstDevice.locator('.device-manager-item-name');
        await expect(name).toBeVisible();

        // Check for device details (type, mount point, available space)
        const details = firstDevice.locator('.device-manager-item-details');
        await expect(details).toBeVisible();
        await expect(details).toContainText('•'); // Should have bullet separators
    });

    test('enabled devices show data directory path', async ({ page }) => {
        await page.goto('/settings');

        await page.locator('.device-manager').waitFor({ state: 'visible', timeout: 5000 });

        // Look for enabled badge with checkmark icon
        const enabledBadge = page.locator('.device-manager-badge--enabled').first();

        if (await enabledBadge.isVisible().catch(() => false)) {
            // Should have checkmark icon
            const icon = enabledBadge.locator('svg');
            await expect(icon).toBeVisible();

            // Should show data directory path
            const deviceItem = enabledBadge.locator('..').locator('..');
            const statusText = deviceItem.locator('.device-manager-item-status');
            await expect(statusText).toBeVisible();
            await expect(statusText).toContainText('data/cirrus');
        }
    });

    test('device statuses API endpoint works', async ({ page }) => {
        // Test the new API endpoint
        const response = await page.request.get('/api/v1/storage/devices/status');
        expect(response.ok()).toBeTruthy();

        const data = await response.json();
        expect(data).toHaveProperty('devices');
        expect(data).toHaveProperty('count');
        expect(Array.isArray(data.devices)).toBeTruthy();

        // Each device should have is_enabled status
        if (data.devices.length > 0) {
            const device = data.devices[0];
            expect(device).toHaveProperty('is_enabled');
            expect(typeof device.is_enabled).toBe('boolean');

            // If enabled, should have data_dir and files_dir
            if (device.is_enabled) {
                expect(device).toHaveProperty('data_dir');
                expect(device).toHaveProperty('files_dir');
                // System devices use ~/Library/Application Support/Autobutler/data (macOS)
                // or ~/autobutler/data (Linux), external devices use .autobutler/data
                expect(device.data_dir).toMatch(/(\.autobutler|Autobutler\/data|autobutler\/data)/);
                expect(device.files_dir).toContain('files');
            }
        }
    });

    test('component reloads on refresh-devices event', async ({ page }) => {
        await page.goto('/settings');

        await page.locator('.device-manager').waitFor({ state: 'visible', timeout: 5000 });

        // The device manager content should listen for refresh-devices event
        const deviceManagerContent = page.locator('#device-manager-content');
        await expect(deviceManagerContent).toHaveAttribute('hx-trigger', /refresh-devices/);
    });

    test('device manager is accessible from settings sidebar', async ({ page }) => {
        await page.goto('/settings');

        // Find storage link in sidebar
        const storageLink = page.locator('.settings-sidebar a[href="#storage"]');
        await expect(storageLink).toBeVisible();

        // Click it
        await storageLink.click();

        // Should scroll to storage section
        await page.waitForTimeout(500); // Wait for scroll animation

        const storageSection = page.locator('#storage');
        await expect(storageSection).toBeInViewport();
    });
});
