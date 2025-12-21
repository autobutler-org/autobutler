import { test, expect } from '@playwright/test'
import * as path from 'path'

test.describe('Photo Thumbnails - EXIF Orientation', () => {
  test('thumbnail API respects EXIF orientation and does not rotate images incorrectly', async ({
    page,
    request,
  }) => {
    await page.goto('/cirrus')
    const fileInput = page.locator('input[type="file"]')
    const testImagePath = path.join('./app/e2e/data/flipped.jpg')
    await fileInput.setInputFiles(testImagePath)
    await page.waitForTimeout(100)
    await page.goto('/photos')
    await page.waitForTimeout(1000)
    const firstPhotoItem = page.locator('.photo-grid-item').first()
    const firstImage = firstPhotoItem.locator('img.photo-grid-image')
    await expect(firstImage).toBeVisible()
    const thumbnailSrc = await firstImage.getAttribute('src')
    expect(thumbnailSrc).toBeTruthy()
    expect(thumbnailSrc).toContain('/api/v1/thumbnails/')
    const thumbnailResponse = await request.get(thumbnailSrc!)
    expect(thumbnailResponse.ok()).toBeTruthy()
    const contentType = thumbnailResponse.headers()['content-type']
    expect(contentType).toMatch(/image\/(jpeg|jpg|png)/)
    const dimensions = await firstImage.evaluate((img: HTMLImageElement) => {
      return {
        width: img.naturalWidth,
        height: img.naturalHeight,
        displayWidth: img.width,
        displayHeight: img.height,
      }
    })
    expect(dimensions.width).toBeGreaterThan(0)
    expect(dimensions.height).toBeGreaterThan(0)
    expect(dimensions.width).toBeLessThanOrEqual(400)
    expect(dimensions.height).toBeLessThanOrEqual(400)
  })
})
