import { test, expect } from '@playwright/test'

test.describe('Settings page', () => {
  test('settings page has storage section', async ({ page }) => {
    await page.goto('/settings')
    // Section exists
    const storageSection = page.locator('#storage.settings-section')
    await expect(storageSection).toBeVisible()
    // Header and title
    const header = storageSection.locator('.settings-section-header h2')
    await expect(header).toHaveText('Storage Devices')
    // Description
    const desc = storageSection.locator('.settings-section-description p')
    await expect(desc).toContainText('Manage which storage devices are enabled')
  })

  test('shows mock card with loading text and badge', async ({ page }) => {
    await page.goto('/settings')
    const mockCard = page.locator('#storage .mock-card')
    await expect(mockCard).toBeVisible()
    await expect(mockCard.locator('.mock-badge')).toHaveText(/mock/i)
    await expect(mockCard.locator('.mock-loading')).toHaveText(
      /loading devices/i,
    )
  })
})
