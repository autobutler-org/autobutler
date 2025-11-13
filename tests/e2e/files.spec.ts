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
        await fileInput.setInputFiles('./tests/e2e/fixtures/sample.txt');

        // Wait for the upload to complete
        await page.waitForTimeout(1000);

        // Verify the file appears in the file list
        const fileName = page.locator('text=sample.txt');
        await expect(fileName).toBeVisible({ timeout: 10000 });
    });
});

test.describe('Files Page - File Interactions', () => {
    test('opens file viewer modal when clicking on a file', async ({ page }) => {
        await page.goto('/files');

        // Click on the uploaded file
        const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
        const fileCell = fileRow.locator('.file-table-cell--clickable');
        await fileCell.click();

        // Verify the file viewer modal is visible
        const fileViewer = page.locator('#file-viewer');
        await expect(fileViewer).toBeVisible();
    });

    test('file viewer modal displays file content', async ({ page }) => {
        await page.goto('/files');

        const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
        const fileCell = fileRow.locator('.file-table-cell--clickable');
        await fileCell.click();

        // Wait for content to load
        await page.waitForTimeout(500);

        // Check that file viewer content exists
        const fileViewerContent = page.locator('#file-viewer-content');
        await expect(fileViewerContent).toBeVisible();
    });

    test('file viewer modal closes with close button', async ({ page }) => {
        await page.goto('/files');

        const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
        const fileCell = fileRow.locator('.file-table-cell--clickable');
        await fileCell.click();

        const fileViewer = page.locator('#file-viewer');
        await expect(fileViewer).toBeVisible();

        // Click the close button
        const closeButton = page.locator('.file-viewer-close');
        await closeButton.click();

        // Modal should be closed
        await expect(fileViewer).not.toBeVisible();
    });

    test('file viewer modal closes with Escape key', async ({ page }) => {
        await page.goto('/files');

        const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
        const fileCell = fileRow.locator('.file-table-cell--clickable');
        await fileCell.click();

        const fileViewer = page.locator('#file-viewer');
        await expect(fileViewer).toBeVisible();

        // Press Escape key
        await page.keyboard.press('Escape');

        // Modal should be closed
        await expect(fileViewer).not.toBeVisible();
    });

    test('context menu opens when clicking trigger button', async ({ page }) => {
        await page.goto('/files');

        const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
        const contextTrigger = fileRow.locator('.context-menu-trigger');

        await contextTrigger.click();
        await page.waitForTimeout(100);

        // Context menu should be visible
        const contextMenu = fileRow.locator('.context-menu');
        await expect(contextMenu).toBeVisible();
    });
});


test.describe('Modal Dialog Behavior', () => {
    test.beforeEach(async ({ page }) => {
        await page.goto('/files');

        // Check if sample.txt exists, if not upload it
        const existingFile = page.locator('tr.file-table-row[data-name="sample.txt"]');
        const fileExists = await existingFile.count() > 0;

        if (!fileExists) {
            const fileInput = page.locator('input[type="file"]');
            await fileInput.setInputFiles('./tests/e2e/fixtures/sample.txt');
            await page.waitForTimeout(1000);
        }

        // Wait for file to appear
        await expect(page.locator('text=sample.txt')).toBeVisible({ timeout: 10000 });
    });

    test('close button should properly close file viewer modal', async ({ page }) => {
        // Open the file viewer
        const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
        const fileCell = fileRow.locator('.file-table-cell--clickable');
        await fileCell.dispatchEvent('click');
        await page.waitForTimeout(100);

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
        await page.waitForTimeout(100);

        const fileViewer = page.locator('#file-viewer');
        await expect(fileViewer).toBeVisible();

        // Click on the content area
        const contentArea = page.locator('#file-viewer-content');
        await contentArea.click();

        // Dialog should still be visible
        await expect(fileViewer).toBeVisible();
    });

    test('clicking outside of rename dialog should close rename modal', async ({ page }) => {
        // Open context menu and click rename
        const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
        await fileRow.locator('.context-menu-trigger').click();
        await page.waitForTimeout(100);

        const renameButton = fileRow.locator('.context-menu-item:has-text("Rename")');
        await renameButton.dispatchEvent('click');

        // Wait for rename dialog to appear
        const renameDialog = page.locator('.ab-rename-overlay');
        await expect(renameDialog).toBeVisible();

        // Get the dialog box (the inner dialog, not the overlay)
        const dialogBox = page.locator('.ab-rename-dialog');
        const dialogBoundingBox = await dialogBox.boundingBox();
        expect(dialogBoundingBox).not.toBeNull();

        await page.mouse.click(10, 10);

        // Dialog should not be visible
        await expect(renameDialog).not.toBeVisible();
        await expect(dialogBox).not.toBeVisible();
    });

    test('close button should properly close rename dialog', async ({ page }) => {
        // Open context menu and click rename
        const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
        await fileRow.locator('.context-menu-trigger').click();
        await page.waitForTimeout(100);

        const renameButton = fileRow.locator('.context-menu-item:has-text("Move/Rename")');
        await renameButton.dispatchEvent('click');

        // Wait for rename dialog to appear
        const renameDialog = page.locator('.ab-rename-overlay');
        await expect(renameDialog).toBeVisible();

        // Click the close button
        const closeButton = page.locator('.ab-rename-close');
        await closeButton.click();

        // Dialog should be closed
        await expect(renameDialog).not.toBeVisible();
    });
});

test.describe('Files Page - File Deletion', () => {
    test('deletes the uploaded file', async ({ page }) => {
        await page.goto('/files');

        const fileRow = page.locator('tr.file-table-row[data-name="sample.txt"]');
        await fileRow.locator('.context-menu-trigger').click();
        await page.waitForTimeout(100);
        await fileRow.locator('.context-menu-item--danger:has-text("Delete")').dispatchEvent('click');

        // Verify file is deleted
        await expect(fileRow).not.toBeVisible();
    });

    test('verifies file is no longer present after deletion', async ({ page }) => {
        await page.goto('/files');
        await expect(page.locator('tr.file-table-row[data-name="sample.txt"]')).not.toBeVisible();
    });
});

test.describe('Files Page - Navigation', () => {
    test('back button navigates from subfolder to parent folder', async ({ page }) => {
        await page.goto('/files');

        // Create a test folder
        const addFolderBtn = page.locator('#add-folder-btn');
        await addFolderBtn.click();
        await page.waitForTimeout(100);
        const folderInput = page.locator('#folder-input');
        await folderInput.fill('test-nav-folder');
        await folderInput.press('Enter');
        await page.waitForTimeout(100);

        // Verify folder was created
        const folderRow = page.locator('tr.file-table-row[data-name="test-nav-folder/"]');
        await expect(folderRow).toBeVisible();

        // Navigate into the folder by clicking on it
        const folderLink = folderRow.locator('a.file-table-link');
        await folderLink.click();
        await page.waitForTimeout(100);

        // Verify we're in the subfolder (URL should change)
        await expect(page).toHaveURL(/\/files\/test-nav-folder/);

        // Find and click the back navigation button
        const backButton = page.locator('#nav-back-btn');
        await expect(backButton).toBeVisible();
        await expect(backButton).not.toBeDisabled();
        await backButton.click();
        await page.waitForTimeout(1000);

        // Verify we're back at the root (URL should be /files)
        await expect(page).toHaveURL(/^.*\/files\/?$/);
    });

    test('back button is disabled at root directory', async ({ page }) => {
        await page.goto('/files');

        const backButton = page.locator('#nav-back-btn');
        await expect(backButton).toBeVisible();
        await expect(backButton).toBeDisabled();
    });
});
