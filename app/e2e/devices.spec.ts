import { test, expect } from '@playwright/test';

test.describe('Devices Page', () => {
  test('loads devices page successfully', async ({ page }) => {
    await page.goto('/devices');
    await expect(page).toHaveTitle(/Autobutler/);
  });

  test('displays devices page header with title and subtitle', async ({
    page,
  }) => {
    await page.goto('/devices');
    const title = page.locator('h1.devices-title');
    await expect(title).toBeVisible();
    await expect(title).toHaveText('Storage Devices');
    const subtitle = page.locator('p.devices-subtitle');
    await expect(subtitle).toBeVisible();
    await expect(subtitle).toContainText('Monitor capacity');
  });

  test('displays device content container', async ({ page }) => {
    await page.goto('/devices');
    const devicesContent = page.locator('#devices-content');
    await expect(devicesContent).toBeVisible();
  });

  test('shows total capacity section when devices are present', async ({
    page,
  }) => {
    await page.goto('/devices');
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
    }
  });
});
