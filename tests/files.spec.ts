import { test, expect } from '@playwright/test';

test.describe('Files Page', () => {
  test('loads files page successfully', async ({ page }) => {
    await page.goto('/files');

    await expect(page).toHaveTitle(/Autobutler/);
    await expect(page.locator('#file-explorer')).toBeVisible();
  });

  test('displays file explorer header with title and space info', async ({ page }) => {
    await page.goto('/files');

    const header = page.locator('.file-explorer-header');
    await expect(header).toBeVisible();

    const title = page.locator('h2.file-explorer-title');
    await expect(title).toBeVisible();
    await expect(title).toHaveText('File Explorer');

    const spaceInfo = page.locator('.file-explorer-space-info');
    await expect(spaceInfo).toBeVisible();
    await expect(spaceInfo).toContainText('Available Space:');
    await expect(spaceInfo).toContainText('GB');
  });

  test('displays file operations buttons (upload, download, delete)', async ({ page }) => {
    await page.goto('/files');

    const fileOperations = page.locator('.file-operations');
    await expect(fileOperations).toBeVisible();

    // Check for operation buttons - they should exist
    const buttons = fileOperations.locator('button');
    const buttonCount = await buttons.count();
    expect(buttonCount).toBeGreaterThan(0);
  });

  test('displays upload progress indicator', async ({ page }) => {
    await page.goto('/files');

    const progressBar = page.locator('progress#file-upload-progress');
    await expect(progressBar).toBeAttached();
    await expect(progressBar).toHaveAttribute('max', '100');
    await expect(progressBar).toHaveAttribute('value', '0');
  });

  test('displays view switcher with three view options', async ({ page }) => {
    await page.goto('/files');

    const viewSwitcher = page.locator('.view-switcher');
    await expect(viewSwitcher).toBeVisible();

    const listViewBtn = viewSwitcher.locator('button[title="List View"]');
    const gridViewBtn = viewSwitcher.locator('button[title="Grid View"]');
    const columnViewBtn = viewSwitcher.locator('button[title="Column View"]');

    await expect(listViewBtn).toBeVisible();
    await expect(gridViewBtn).toBeVisible();
    await expect(columnViewBtn).toBeVisible();
  });

  test('list view button is active by default', async ({ page }) => {
    await page.goto('/files');

    const listViewBtn = page.locator('button[title="List View"]');
    const classes = await listViewBtn.getAttribute('class');

    expect(classes).toContain('btn--primary');
  });

  test('displays breadcrumb navigation', async ({ page }) => {
    await page.goto('/files');

    // Look for the breadcrumbs nav element
    const breadcrumb = page.locator('nav#breadcrumbs');
    await expect(breadcrumb).toBeVisible();
  });

  test('displays file explorer view content area', async ({ page }) => {
    await page.goto('/files');

    const viewContent = page.locator('#file-explorer-view-content');
    await expect(viewContent).toBeVisible();
  });

  test('switching to grid view via query parameter', async ({ page }) => {
    await page.goto('/files?view=grid');

    // Verify page loads successfully
    await expect(page).toHaveTitle(/Autobutler/);
    await expect(page.locator('#file-explorer')).toBeVisible();
  });

  test('switching to column view via query parameter', async ({ page }) => {
    await page.goto('/files?view=column');

    // Verify page loads successfully
    await expect(page).toHaveTitle(/Autobutler/);
    await expect(page.locator('#file-explorer')).toBeVisible();
  });

  test('file viewer dialog exists', async ({ page }) => {
    await page.goto('/files');

    const fileViewer = page.locator('#file-viewer');
    await expect(fileViewer).toBeAttached();
  });

  test('explorer context menu exists', async ({ page }) => {
    await page.goto('/files');

    // Context menu should exist in the DOM (may be hidden initially)
    // There can be multiple context menus (one per file), so just check at least one exists
    const contextMenus = page.locator('ul.context-menu');
    const count = await contextMenus.count();
    expect(count).toBeGreaterThan(0);
  });

  test('file explorer status area exists', async ({ page }) => {
    await page.goto('/files');

    const statusArea = page.locator('#file-explorer-status');
    await expect(statusArea).toBeAttached();
  });

  test('loads Selecto library for multi-select', async ({ page }) => {
    await page.goto('/files');

    const selectoScript = page.locator('script[src="/public/vendor/selecto/selecto.min.js"]');
    await expect(selectoScript).toBeAttached();
  });

  test('loads file explorer script', async ({ page }) => {
    await page.goto('/files');

    const script = page.locator('script[src="/public/scripts/file_explorer.js"]');
    await expect(script).toBeAttached();
  });

  test('file explorer has HTMX support for dynamic updates', async ({ page }) => {
    await page.goto('/files');

    // Check for HTMX attributes on interactive elements
    const htmxElements = page.locator('[hx-get], [hx-post], [hx-delete]');
    const count = await htmxElements.count();

    expect(count).toBeGreaterThan(0);
  });

  test('drag and drop area exists for file uploads', async ({ page }) => {
    await page.goto('/files');

    const dndArea = page.locator('#file-upload-area, [class*="upload"]');
    const count = await dndArea.count();

    // Upload area should exist
    expect(count).toBeGreaterThan(0);
  });
});

test.describe('Files Page - File Upload', () => {
  test('uploads a text file through file input', async ({ page }) => {
    await page.goto('/files');

    // Find the file input element
    const fileInput = page.locator('input[type="file"]');

    // Upload the test file
    await fileInput.setInputFiles('./tests/fixtures/sample.txt');

    // Wait for the upload to complete (progress bar or file to appear)
    await page.waitForTimeout(1000);

    // Verify the file appears in the file list
    const fileName = page.locator('text=sample.txt');
    await expect(fileName).toBeVisible({ timeout: 10000 });
  });

  test('uploads a JSON file and verifies it appears in list', async ({ page }) => {
    await page.goto('/files');

    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('./tests/fixtures/data.json');

    await page.waitForTimeout(1000);

    const fileName = page.locator('text=data.json');
    await expect(fileName).toBeVisible({ timeout: 10000 });
  });

  test('uploads a CSV file', async ({ page }) => {
    await page.goto('/files');

    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('./tests/fixtures/users.csv');

    await page.waitForTimeout(1000);

    const fileName = page.locator('text=users.csv');
    await expect(fileName).toBeVisible({ timeout: 10000 });
  });

  test('uploads a markdown file', async ({ page }) => {
    await page.goto('/files');

    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('./tests/fixtures/test-document.md');

    await page.waitForTimeout(1000);

    const fileName = page.locator('text=test-document.md');
    await expect(fileName).toBeVisible({ timeout: 10000 });
  });

  test('uploads an HTML file', async ({ page }) => {
    await page.goto('/files');

    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles('./tests/fixtures/sample.html');

    await page.waitForTimeout(1000);

    const fileName = page.locator('text=sample.html');
    await expect(fileName).toBeVisible({ timeout: 10000 });
  });

  test('uploads multiple files at once', async ({ page }) => {
    await page.goto('/files');

    const fileInput = page.locator('input[type="file"]');

    // Upload multiple files
    await fileInput.setInputFiles([
      './tests/fixtures/sample.txt',
      './tests/fixtures/data.json'
    ]);

    await page.waitForTimeout(2000);

    // Verify both files appear
    await expect(page.locator('text=sample.txt')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=data.json')).toBeVisible({ timeout: 10000 });
  });
});
