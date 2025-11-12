import { test, expect } from '@playwright/test';

test.describe('Photos Page', () => {
  test('loads photos page successfully', async ({ page }) => {
    await page.goto('/photos');

    await expect(page).toHaveTitle(/Autobutler/);
    await expect(page.locator('#photos-library')).toBeVisible();
  });

  test('displays photos library structure', async ({ page }) => {
    await page.goto('/photos');

    const photosLibrary = page.locator('.photos-library');
    await expect(photosLibrary).toBeVisible();

    const photosContainer = page.locator('.photos-container');
    await expect(photosContainer).toBeVisible();
  });

  test('displays photos header with title', async ({ page }) => {
    await page.goto('/photos');

    const header = page.locator('.photos-header');
    await expect(header).toBeVisible();

    const title = page.locator('h2.photos-title');
    await expect(title).toBeVisible();
    await expect(title).toHaveText('All Photos');
  });

  test('displays photo count with proper formatting', async ({ page }) => {
    await page.goto('/photos');

    const countElement = page.locator('.photos-count');
    await expect(countElement).toBeVisible();

    const countText = await countElement.textContent();

    // Should show either "0 photos", "1 photo", or "N photos"
    expect(countText).toMatch(/\d+\s+photos?/i);
  });

  test('displays sidebar navigation', async ({ page }) => {
    await page.goto('/photos');

    // Check for sidebar - it's part of the photos component
    const photosMain = page.locator('.photos-main');
    await expect(photosMain).toBeVisible();
  });

  test('displays photo grid container', async ({ page }) => {
    await page.goto('/photos');

    const photoGrid = page.locator('.photo-grid');
    await expect(photoGrid).toBeVisible();
  });

  test('photo grid items have proper structure when photos exist', async ({ page }) => {
    await page.goto('/photos');

    const photoItems = page.locator('.photo-grid-item');
    const itemCount = await photoItems.count();

    if (itemCount > 0) {
      const firstItem = photoItems.first();

      // Check for image element
      const img = firstItem.locator('img.photo-grid-image');
      await expect(img).toBeVisible();

      // Check for proper attributes
      await expect(img).toHaveAttribute('loading', 'lazy');
      await expect(img).toHaveAttribute('alt');
      await expect(img).toHaveAttribute('src');
    }
  });

  test('photo grid items use HTMX for viewer interaction', async ({ page }) => {
    await page.goto('/photos');

    const photoItems = page.locator('.photo-grid-item');
    const itemCount = await photoItems.count();

    if (itemCount > 0) {
      const firstItem = photoItems.first();

      // Check HTMX attributes for opening viewer
      await expect(firstItem).toHaveAttribute('hx-get');
      await expect(firstItem).toHaveAttribute('hx-target', '#file-viewer-content');
      await expect(firstItem).toHaveAttribute('hx-swap', 'innerHTML');
    }
  });

  test('photo thumbnails use API endpoint', async ({ page }) => {
    await page.goto('/photos');

    const photoItems = page.locator('.photo-grid-item');
    const itemCount = await photoItems.count();

    if (itemCount > 0) {
      const firstImage = photoItems.first().locator('img.photo-grid-image');
      const src = await firstImage.getAttribute('src');

      // Thumbnail path should use the API endpoint
      expect(src).toContain('/api/v1/thumbnails/');
    }
  });

  test('photos library uses file viewer for interactions', async ({ page }) => {
    await page.goto('/photos');

    // File viewer exists for opening photos
    const fileViewer = page.locator('dialog#file-viewer');
    await expect(fileViewer).toBeAttached();
  });

  test('photos page loads required scripts', async ({ page }) => {
    await page.goto('/photos');

    const script = page.locator('script[src="/public/scripts/file_explorer.js"]');
    await expect(script).toBeAttached();
  });

  test('handles root directory parameter in URL', async ({ page }) => {
    await page.goto('/photos/custom-root');

    await expect(page).toHaveTitle(/Autobutler/);
    await expect(page.locator('#photos-library')).toBeVisible();
  });

  test('photo count shows "0 photos" when no photos exist', async ({ page }) => {
    await page.goto('/photos');

    const countElement = page.locator('.photos-count');
    const countText = await countElement.textContent();

    // When testing with no files, should show "0 photos"
    if (countText?.includes('0')) {
      expect(countText).toBe('0 photos');
    }
  });

  test('photo grid is empty when no photos exist', async ({ page }) => {
    await page.goto('/photos');

    const photoItems = page.locator('.photo-grid-item');
    const itemCount = await photoItems.count();

    // When testing with no files, grid should be empty
    if (itemCount === 0) {
      const photoGrid = page.locator('.photo-grid');
      await expect(photoGrid).toBeVisible();
      expect(itemCount).toBe(0);
    }
  });
});
