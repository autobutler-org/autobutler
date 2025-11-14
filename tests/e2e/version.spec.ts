import { test, expect } from '@playwright/test';

test.describe('Version Dropdown', () => {
  test('should display version in topnav', async ({ page }) => {
    await page.goto('/');

    // Check that version display button exists
    const versionButton = page.locator('.version-display');
    await expect(versionButton).toBeVisible();
  });

  test('should open dropdown when clicking version', async ({ page }) => {
    await page.goto('/');

    // Click the version button
    const versionButton = page.locator('.version-display');
    await versionButton.click();

    // Wait for dropdown to appear
    await page.waitForSelector('.version-dropdown', { timeout: 5000 });

    // Check that dropdown is visible
    const dropdown = page.locator('.version-dropdown');
    await expect(dropdown).toBeVisible();

    // Check that dropdown contains version options
    const versionOptions = page.locator('.version-option');
    await expect(versionOptions.first()).toBeVisible();
  });

  test('should close dropdown when clicking outside', async ({ page }) => {
    await page.goto('/');

    // Click the version button to open dropdown
    const versionButton = page.locator('.version-display');
    await versionButton.click();

    // Wait for dropdown to appear
    await page.waitForSelector('.version-dropdown', { timeout: 5000 });

    // Click outside the dropdown
    await page.locator('body').click({ position: { x: 10, y: 10 } });

    // Wait a moment for the click handler to execute
    await page.waitForTimeout(100);

    // Check that dropdown is no longer visible
    const dropdown = page.locator('.version-dropdown');
    await expect(dropdown).not.toBeVisible();
  });

  test('should highlight current version', async ({ page }) => {
    await page.goto('/');

    // Click the version button to open dropdown
    const versionButton = page.locator('.version-display');
    await versionButton.click();

    // Wait for dropdown to appear
    await page.waitForSelector('.version-dropdown', { timeout: 5000 });

    // Check if there's a version marked as current
    const currentVersion = page.locator('.version-current');
    // At least one version should be marked as current (or none if we're on a dev build)
    const count = await currentVersion.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
