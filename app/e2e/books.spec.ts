import { expect, test } from '@playwright/test'

test.describe('Books Page', () => {
  test('displays books library header', async ({ page }) => {
    await page.goto('/books')
    const title = page.locator('.library-title')
    await expect(title).toBeVisible()
    await expect(title).toHaveText('Library')
  })

  test('displays book count with proper formatting', async ({ page }) => {
    await page.goto('/books')
    const countElement = page.locator('.library-subtitle')
    await expect(countElement).toBeVisible()
    const countText = await countElement.textContent()
    expect(countText).toMatch(/\d+\s+books?/i)
  })

  test('shows empty state when no books exist', async ({ page }) => {
    await page.goto('/books')
    const bookCount = await page.locator('.book-card').count()
    if (bookCount === 0) {
      const emptyState = page.locator('.books-empty')
      await expect(emptyState).toBeVisible()
      const emptyTitle = page.locator('.books-empty h2')
      await expect(emptyTitle).toBeVisible()
      await expect(emptyTitle).toHaveText('No books found')
      const emptyMessage = page.locator('.books-empty p')
      await expect(emptyMessage).toBeVisible()
      await expect(emptyMessage).toContainText('Add PDF or EPUB files')
      const emptyIcon = page.locator('.books-empty svg')
      await expect(emptyIcon).toBeVisible()
    }
  })

  test('displays books grid when books exist', async ({ page }) => {
    await page.goto('/books')
    const bookCount = await page.locator('.book-card').count()
    if (bookCount > 0) {
      const grid = page.locator('.books-grid')
      await expect(grid).toBeVisible()
      const firstBook = page.locator('.book-card').first()
      await expect(firstBook).toBeVisible()
    }
  })
})
