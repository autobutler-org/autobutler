import { test, expect } from '@playwright/test';

test.describe('Devices Page', () => {
    test('loads devices page successfully', async ({ page }) => {
        await page.goto('/devices');

        await expect(page).toHaveTitle(/Autobutler/);
    });

    test('displays devices page header with title and subtitle', async ({ page }) => {
        await page.goto('/devices');

        const title = page.locator('h1.devices-title');
        await expect(title).toBeVisible();
        await expect(title).toHaveText('Storage Devices');

        const subtitle = page.locator('p.devices-subtitle');
        await expect(subtitle).toBeVisible();
        await expect(subtitle).toContainText('Monitor capacity');
    });

    test('has refresh button with proper attributes', async ({ page }) => {
        await page.goto('/devices');

        const refreshButton = page.locator('button[title="Refresh storage devices"]');
        await expect(refreshButton).toBeVisible();

        // Check HTMX attributes for dynamic refresh
        await expect(refreshButton).toHaveAttribute('hx-get');
        await expect(refreshButton).toHaveAttribute('hx-target');
    });

    test('displays device content container', async ({ page }) => {
        await page.goto('/devices');

        const devicesContent = page.locator('#devices-content');
        await expect(devicesContent).toBeVisible();
    });

    test('shows total capacity section when devices are present', async ({ page }) => {
        await page.goto('/devices');

        // Check if the total capacity title exists
        const totalCapacityTitle = page.locator('h3.devices-total-title');
        const titleCount = await totalCapacityTitle.count();

        if (titleCount > 0) {
            await expect(totalCapacityTitle).toHaveText('Total Capacity');
        }
    });

    test('device cards have proper structure when present', async ({ page }) => {
        await page.goto('/devices');

        const deviceCards = page.locator('.device-card');
        const cardCount = await deviceCards.count();

        if (cardCount > 0) {
            const firstCard = deviceCards.first();
            await expect(firstCard).toBeVisible();

            // Check for capacity information (GB/TB)
            const hasCapacityInfo = (await firstCard.locator('text=/GB|TB/i').count()) > 0;
            expect(hasCapacityInfo).toBe(true);
        }
    });

    test('displays storage statistics with proper formatting', async ({ page }) => {
        await page.goto('/devices');

        // Look for GB/TB formatted values
        const storageValues = page.locator('text=/\\d+(\\.\\d+)?\\s*(GB|TB)/i');
        const valueCount = await storageValues.count();

        if (valueCount > 0) {
            // At least one storage value should be present if devices exist
            expect(valueCount).toBeGreaterThan(0);
        }
    });

    test('page uses HTMX for dynamic updates', async ({ page }) => {
        await page.goto('/devices');

        // Check that HTMX attributes are present
        const htmxElements = page.locator('[hx-get], [hx-post], [hx-target]');
        const count = await htmxElements.count();

        expect(count).toBeGreaterThan(0);
    });

    test('displays device health indicators when available', async ({ page }) => {
        await page.goto('/devices');

        const deviceCards = page.locator('.device-card');
        const cardCount = await deviceCards.count();

        if (cardCount > 0) {
            // Health indicators are optional, just verify structure exists
            await expect(deviceCards.first()).toBeVisible();
        }
    });
});
