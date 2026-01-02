import { test, expect } from '@playwright/test';

test.describe('Photos Page', () => {
  test('loads photos page successfully', async ({ page }) => {
    await page.goto('/photos');
    await expect(page).toHaveTitle(/Autobutler/);
    await expect(page.locator('.photos-grid-container')).toBeVisible();
  });

  test('displays photos header with title', async ({ page }) => {
    await page.goto('/photos');
    const title = page.locator('.library-title');
    await expect(title).toBeVisible();
    await expect(title).toHaveText('All Photos');
  });

  test('displays photo count with proper formatting', async ({ page }) => {
    await page.goto('/photos');
    const countElement = page.locator('.library-subtitle');
    await expect(countElement).toBeVisible();
    const countText = await countElement.textContent();
    expect(countText).toMatch(/\d+\s+photos?/i);
  });

  test('displays photo grid container', async ({ page }) => {
    await page.goto('/photos');
    const photoGrid = page.locator('.photo-grid');
    await expect(photoGrid).toBeVisible();
  });

  test('photo grid items have proper structure when photos exist', async ({
    page,
  }) => {
    await page.goto('/photos');
    const photoItems = page.locator('.photo-grid-item');
    const count = await photoItems.count();
    if (count > 0) {
      const firstItem = photoItems.first();
      await expect(firstItem).toBeVisible();
      const image = firstItem.locator('img.photo-grid-image');
      await expect(image).toBeVisible();
    }
  });
});
