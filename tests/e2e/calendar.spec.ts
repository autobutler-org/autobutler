import { test, expect } from '@playwright/test';

test.describe('Calendar Page', () => {
    test('loads calendar page successfully', async ({ page }) => {
        await page.goto('/calendar');

        await expect(page).toHaveTitle(/Autobutler/);
        await expect(page.locator('#calendar')).toBeVisible();
    });

    test('displays calendar navigation header with prev/next buttons', async ({ page }) => {
        await page.goto('/calendar');

        const navHeader = page.locator('.calendar-header-nav');
        await expect(navHeader).toBeVisible();

        // Check for navigation buttons
        const prevButton = page.locator('button.calendar-nav-btn--prev');
        const nextButton = page.locator('button.calendar-nav-btn--next');

        await expect(prevButton).toBeVisible();
        await expect(nextButton).toBeVisible();
        await expect(prevButton).toHaveAttribute('aria-label', 'Previous month');
        await expect(nextButton).toHaveAttribute('aria-label', 'Next month');
    });

    test('displays current month and year in header', async ({ page }) => {
        await page.goto('/calendar');

        const currentDate = new Date();
        const monthName = currentDate.toLocaleString('default', { month: 'long' });
        const year = currentDate.getFullYear();

        const title = page.locator('h1.calendar-title');
        await expect(title).toBeVisible();

        const titleText = await title.textContent();
        expect(titleText).toContain(monthName);
        expect(titleText).toContain(year.toString());
    });

    test('renders calendar table with proper structure', async ({ page }) => {
        await page.goto('/calendar');

        const calendarTable = page.locator('table.calendar-table');
        await expect(calendarTable).toBeVisible();

        // Check for header row with weekday names
        const headerRow = calendarTable.locator('thead tr');
        await expect(headerRow).toBeVisible();

        const headers = calendarTable.locator('thead th.calendar-header');
        await expect(headers).toHaveCount(7); // 7 days of the week

        // Check for body with calendar rows
        const body = calendarTable.locator('tbody.calendar-body');
        await expect(body).toBeVisible();

        const rows = body.locator('tr.calendar-row');
        const rowCount = await rows.count();
        expect(rowCount).toBeGreaterThanOrEqual(4); // At least 4 weeks
        expect(rowCount).toBeLessThanOrEqual(6); // At most 6 weeks
    });

    test('each day cell has proper structure and data attributes', async ({ page }) => {
        await page.goto('/calendar');

        // Get the first day cell
        const firstDayCell = page
            .locator('table.calendar-table tbody tr')
            .first()
            .locator('td')
            .first();
        await expect(firstDayCell).toBeVisible();

        // Check for data attributes (year, month, day)
        const hasDataDay = await firstDayCell.getAttribute('data-day');
        expect(hasDataDay).not.toBeNull();
    });

    test('navigates to specific month via query parameters', async ({ page }) => {
        await page.goto('/calendar?year=2025&month=January');

        const title = page.locator('h1.calendar-title');
        await expect(title).toBeVisible();

        const titleText = await title.textContent();
        expect(titleText).toContain('January');
        expect(titleText).toContain('2025');
    });

    test('new event dialog exists and is initially hidden', async ({ page }) => {
        await page.goto('/calendar');

        const dialog = page.locator('dialog#new-event-dialog');
        await expect(dialog).toBeAttached();

        // Dialog should not be visible initially
        const isVisible = await dialog.isVisible();
        expect(isVisible).toBe(false);
    });

    test('calendar uses HTMX for navigation', async ({ page }) => {
        await page.goto('/calendar');

        const prevButton = page.locator('button.calendar-nav-btn--prev');

        // Check for HTMX attributes
        await expect(prevButton).toHaveAttribute('hx-get');
        await expect(prevButton).toHaveAttribute('hx-target', '#calendar');
        await expect(prevButton).toHaveAttribute('hx-swap', 'outerHTML');
    });

    test('calendar script is loaded', async ({ page }) => {
        await page.goto('/calendar');

        // Check that the calendar script tag is present
        const script = page.locator('script[src="/public/scripts/calendar.js"]');
        await expect(script).toBeAttached();
    });

    test('clicking delete button in event dialog works and does not open other dialogs', async ({
        page,
    }) => {
        await page.goto('/calendar');

        // Find a calendar event (if it exists)
        const eventItem = page.locator('.calendar-event-item').first();

        // If no events exist, skip this test
        const eventCount = await page.locator('.calendar-event-item').count();
        if (eventCount === 0) {
            test.skip();
            return;
        }

        // Click on the event to open its dialog
        await eventItem.click();

        // Wait for the dialog to open
        const dialog = page.locator('dialog[id^="event-dialog-"]').first();
        await expect(dialog).toBeVisible();

        // Check if the delete button exists (it won't exist for new events)
        const deleteButton = dialog.locator('.event-editor-delete-btn');
        const deleteButtonCount = await deleteButton.count();

        if (deleteButtonCount === 0) {
            // This is a new event dialog, not an existing event
            test.skip();
            return;
        }

        // Track open dialogs before clicking delete
        const dialogsBefore = await page.locator('dialog[open]').count();

        // Click the delete button - verify it works and doesn't trigger other actions
        await deleteButton.click();

        // After delete, the calendar should be updated and dialog closed
        // Wait a moment for the deletion to process
        await page.waitForTimeout(200);

        // Verify that no additional dialogs were opened due to click propagation
        const dialogsAfter = await page.locator('dialog[open]').count();
        expect(dialogsAfter).toBeLessThanOrEqual(dialogsBefore);
    });

    test('clicking backdrop closes event dialog', async ({ page }) => {
        await page.goto('/calendar');

        // Find a calendar event (if it exists)
        const eventItem = page.locator('.calendar-event-item').first();

        // If no events exist, skip this test
        const eventCount = await page.locator('.calendar-event-item').count();
        if (eventCount === 0) {
            test.skip();
            return;
        }

        // Click on the event to open its dialog
        await eventItem.click();

        // Wait for the dialog to open
        const dialog = page.locator('dialog[id^="event-dialog-"]').first();
        await expect(dialog).toBeVisible();

        // Click on the backdrop (the dialog element itself, not its content)
        // This simulates clicking outside the modal content
        await dialog.click({ position: { x: 5, y: 5 } });

        // Dialog should now be closed
        const isVisible = await dialog.isVisible();
        expect(isVisible).toBe(false);
    });

    test('close button in event dialog works', async ({ page }) => {
        await page.goto('/calendar');

        // Find a calendar event (if it exists)
        const eventItem = page.locator('.calendar-event-item').first();

        // If no events exist, skip this test
        const eventCount = await page.locator('.calendar-event-item').count();
        if (eventCount === 0) {
            test.skip();
            return;
        }

        // Click on the event to open its dialog
        await eventItem.click();

        // Wait for the dialog to open
        const dialog = page.locator('dialog[id^="event-dialog-"]').first();
        await expect(dialog).toBeVisible();

        // Find and click the close button
        const closeButton = dialog.locator('.event-editor-close-btn');
        await closeButton.click();

        // Dialog should now be closed
        const isVisible = await dialog.isVisible();
        expect(isVisible).toBe(false);
    });
});
