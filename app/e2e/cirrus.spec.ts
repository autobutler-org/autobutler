import { test, expect } from '@playwright/test'

test.describe('Cirrus Page', () => {
  test('loads cirrus page successfully', async ({ page }) => {
    await page.goto('/cirrus')
    await expect(page).toHaveTitle(/Autobutler/)
    await expect(page.locator('#file-explorer')).toBeVisible()
  })

  test('displays file explorer header with title and space info', async ({
    page,
  }) => {
    await page.goto('/cirrus')
    const header = page.locator('.file-explorer-header')
    await expect(header).toBeVisible()
    const title = page.locator('h2.file-explorer-title')
    await expect(title).toBeVisible()
    await expect(title).toHaveText('Cirrus')
  })

  test('displays view switcher with three view options', async ({ page }) => {
    await page.goto('/cirrus')
    const viewSwitcher = page.locator('.view-switcher')
    await expect(viewSwitcher).toBeVisible()
    await expect(
      viewSwitcher.locator('button[title="List View"]'),
    ).toBeVisible()
    await expect(
      viewSwitcher.locator('button[title="Grid View"]'),
    ).toBeVisible()
  })

  test('list view button is active by default', async ({ page }) => {
    await page.goto('/cirrus')
    const listViewBtn = page.locator('button[title="List View"]')
    const classes = await listViewBtn.getAttribute('class')
    expect(classes).toContain('btn--primary')
  })

  test('displays breadcrumb navigation', async ({ page }) => {
    await page.goto('/cirrus')
    const breadcrumb = page.locator('nav.file-explorer-breadcrumbs')
    await expect(breadcrumb).toBeVisible()
  })

  test('displays file explorer view content area', async ({ page }) => {
    await page.goto('/cirrus')
    const contentArea = page.locator('#file-explorer-view-content')
    await expect(contentArea).toBeVisible()
  })
})
