import { test, expect } from '@playwright/test';

test.describe('Version Dropdown', () => {
  test('should display version in topnav', async ({ page }) => {
    await page.goto('/');
    const versionButton = page.locator('.version-display');
    await expect(versionButton).toBeVisible();
  });

  test.skip('should open dropdown when clicking version', async ({ page }) => {
    await page.goto('/');
    const versionButton = page.locator('.version-display');
    await versionButton.click();
    await page.waitForSelector('.version-dropdown', { timeout: 500 });
    const dropdown = page.locator('.version-dropdown');
    await expect(dropdown).toBeVisible();
    const versionOptions = page.locator('.version-dropdown-item');
    await expect(versionOptions.first()).toBeVisible();
  });

  test.skip('should close dropdown when clicking outside', async ({ page }) => {
    await page.goto('/');
    const versionButton = page.locator('.version-display');
    await versionButton.click();
    await page.waitForSelector('.version-dropdown', { timeout: 500 });
    await page.locator('body').click({ position: { x: 10, y: 10 } });
    await page.waitForTimeout(100);
    const dropdown = page.locator('.version-dropdown');
    await expect(dropdown).not.toBeVisible();
  });
});
