import { test, expect } from '@playwright/test';

test.describe('Modal Dialog Behavior', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/files');

    // Upload a test file first
    const fileInput = page.locator('#file-upload-input');
    await fileInput.setInputFiles('./tests/fixtures/sample.txt');

    // Wait for file to appear
    await expect(page.locator('text=sample.txt')).toBeVisible({ timeout: 10000 });
  });

  test('clicking on dialog backdrop should NOT close file viewer modal', async ({ page }) => {
    // Open the file viewer by clicking on a file
    const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
    const fileCell = fileRow.locator('.file-table-cell--clickable');
    await fileCell.click();

    // Wait for the dialog to open
    const fileViewer = page.locator('#file-viewer');
    await expect(fileViewer).toBeVisible();

    // Get the dialog element's bounding box
    const dialogBox = await fileViewer.boundingBox();
    expect(dialogBox).not.toBeNull();

    // Click on the backdrop (outside the dialog, but within the viewport)
    // Click to the left of the dialog
    await page.mouse.click(dialogBox!.x - 20, dialogBox!.y + dialogBox!.height / 2);

    // Dialog should still be visible
    await expect(fileViewer).toBeVisible();

    // Click to the right of the dialog
    await page.mouse.click(dialogBox!.x + dialogBox!.width + 20, dialogBox!.y + dialogBox!.height / 2);

    // Dialog should still be visible
    await expect(fileViewer).toBeVisible();

    // Click above the dialog
    await page.mouse.click(dialogBox!.x + dialogBox!.width / 2, dialogBox!.y - 20);

    // Dialog should still be visible
    await expect(fileViewer).toBeVisible();

    // Click below the dialog
    await page.mouse.click(dialogBox!.x + dialogBox!.width / 2, dialogBox!.y + dialogBox!.height + 20);

    // Dialog should still be visible
    await expect(fileViewer).toBeVisible();
  });

  test('clicking on dialog border/padding area should NOT close file viewer modal', async ({ page }) => {
    // Open the file viewer
    const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
    const fileCell = fileRow.locator('.file-table-cell--clickable');
    await fileCell.click();

    const fileViewer = page.locator('#file-viewer');
    await expect(fileViewer).toBeVisible();

    // Get dialog dimensions
    const dialogBox = await fileViewer.boundingBox();
    expect(dialogBox).not.toBeNull();

    // Click near the edge of the dialog (on the border/padding area)
    // Top-left corner area
    await page.mouse.click(dialogBox!.x + 5, dialogBox!.y + 5);
    await expect(fileViewer).toBeVisible();

    // Top-right corner area
    await page.mouse.click(dialogBox!.x + dialogBox!.width - 5, dialogBox!.y + 5);
    await expect(fileViewer).toBeVisible();

    // Bottom-left corner area
    await page.mouse.click(dialogBox!.x + 5, dialogBox!.y + dialogBox!.height - 5);
    await expect(fileViewer).toBeVisible();

    // Bottom-right corner area
    await page.mouse.click(dialogBox!.x + dialogBox!.width - 5, dialogBox!.y + dialogBox!.height - 5);
    await expect(fileViewer).toBeVisible();
  });

  test('close button should properly close file viewer modal', async ({ page }) => {
    // Open the file viewer
    const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
    const fileCell = fileRow.locator('.file-table-cell--clickable');
    await fileCell.click();

    const fileViewer = page.locator('#file-viewer');
    await expect(fileViewer).toBeVisible();

    // Click the close button
    const closeButton = page.locator('.file-viewer-close');
    await closeButton.click();

    // Dialog should be closed (not visible)
    await expect(fileViewer).not.toBeVisible();
  });

  test('clicking within dialog content should NOT close file viewer modal', async ({ page }) => {
    // Open the file viewer
    const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
    const fileCell = fileRow.locator('.file-table-cell--clickable');
    await fileCell.click();

    const fileViewer = page.locator('#file-viewer');
    await expect(fileViewer).toBeVisible();

    // Click on the content area
    const contentArea = page.locator('#file-viewer-content');
    await contentArea.click();

    // Dialog should still be visible
    await expect(fileViewer).toBeVisible();
  });

  test('clicking on rename dialog backdrop should NOT close rename modal', async ({ page }) => {
    // Open context menu and click rename
    const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
    await fileRow.locator('.context-menu-trigger').click();
    await page.waitForTimeout(100);

    const renameButton = fileRow.locator('.context-menu-item:has-text("Rename/Move")');
    await renameButton.click();

    // Wait for rename dialog to appear
    const renameDialog = page.locator('.ab-rename-overlay');
    await expect(renameDialog).toBeVisible();

    // Get the dialog box (the inner dialog, not the overlay)
    const dialogBox = page.locator('.ab-rename-dialog');
    const dialogBoundingBox = await dialogBox.boundingBox();
    expect(dialogBoundingBox).not.toBeNull();

    // Click on the overlay backdrop (outside the dialog box)
    await page.mouse.click(10, 10); // Top-left corner of viewport, which should be on the overlay

    // Dialog should still be visible
    await expect(renameDialog).toBeVisible();
    await expect(dialogBox).toBeVisible();
  });

  test('close button should properly close rename dialog', async ({ page }) => {
    // Open context menu and click rename
    const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
    await fileRow.locator('.context-menu-trigger').click();
    await page.waitForTimeout(100);

    const renameButton = fileRow.locator('.context-menu-item:has-text("Rename/Move")');
    await renameButton.click();

    // Wait for rename dialog to appear
    const renameDialog = page.locator('.ab-rename-overlay');
    await expect(renameDialog).toBeVisible();

    // Click the close button
    const closeButton = page.locator('.ab-rename-close');
    await closeButton.click();

    // Dialog should be closed
    await expect(renameDialog).not.toBeVisible();
  });

  test.afterEach(async ({ page }) => {
    // Clean up: delete the test file if it exists
    const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
    const fileExists = await fileRow.count() > 0;

    if (fileExists) {
      await fileRow.locator('.context-menu-trigger').click();
      await page.waitForTimeout(100);
      await fileRow.locator('.context-menu-item--danger:has-text("Delete")').dispatchEvent('click');
    }
  });
});
