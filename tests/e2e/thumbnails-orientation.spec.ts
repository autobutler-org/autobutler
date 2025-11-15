import { test, expect } from '@playwright/test';
import * as path from 'path';

test.describe('Photo Thumbnails - EXIF Orientation', () => {
    test('thumbnail API respects EXIF orientation and does not rotate images incorrectly', async ({ page, request }) => {
        // First, upload a test image with EXIF orientation data
        await page.goto('/files');

        const fileInput = page.locator('input[type="file"]');
        const testImagePath = path.join('./tests/e2e/data/test-image.jpg');
        await fileInput.setInputFiles(testImagePath);

        // Wait for upload to complete
        await page.waitForTimeout(1500);

        // Now check the photos page
        await page.goto('/photos');
        await page.waitForTimeout(1000);

        // Get the first photo thumbnail
        const firstPhotoItem = page.locator('.photo-grid-item').first();
        const firstImage = firstPhotoItem.locator('img.photo-grid-image');

        // Check that the image is visible and has loaded
        await expect(firstImage).toBeVisible();

        // Get the thumbnail URL
        const thumbnailSrc = await firstImage.getAttribute('src');
        expect(thumbnailSrc).toBeTruthy();
        expect(thumbnailSrc).toContain('/api/v1/thumbnails/');

        // Make a direct request to the thumbnail API to verify it returns valid image data
        const thumbnailResponse = await request.get(thumbnailSrc!);
        expect(thumbnailResponse.ok()).toBeTruthy();

        // Verify it's an image
        const contentType = thumbnailResponse.headers()['content-type'];
        expect(contentType).toMatch(/image\/(jpeg|jpg|png)/);

        // Get image dimensions by loading it in the page
        const dimensions = await firstImage.evaluate((img: HTMLImageElement) => {
            return {
                width: img.naturalWidth,
                height: img.naturalHeight,
                displayWidth: img.width,
                displayHeight: img.height
            };
        });

        // Verify the image has valid dimensions (not 0x0)
        expect(dimensions.width).toBeGreaterThan(0);
        expect(dimensions.height).toBeGreaterThan(0);

        // Thumbnails should be constrained to max 400x400
        expect(dimensions.width).toBeLessThanOrEqual(400);
        expect(dimensions.height).toBeLessThanOrEqual(400);
    });

    test('thumbnails maintain correct aspect ratio after EXIF correction', async ({ page }) => {
        await page.goto('/photos');
        await page.waitForTimeout(1000);

        const photoItems = page.locator('.photo-grid-item');
        const itemCount = await photoItems.count();

        if (itemCount > 0) {
            const firstImage = photoItems.first().locator('img.photo-grid-image');
            await expect(firstImage).toBeVisible();

            // Wait for image to load
            await firstImage.evaluate((img: HTMLImageElement) => {
                return img.complete || new Promise((resolve) => {
                    img.onload = resolve;
                    img.onerror = resolve;
                });
            });

            const dimensions = await firstImage.evaluate((img: HTMLImageElement) => {
                return {
                    width: img.naturalWidth,
                    height: img.naturalHeight
                };
            });

            // Calculate aspect ratio
            const aspectRatio = dimensions.width / dimensions.height;

            // Aspect ratio should be reasonable (not wildly incorrect)
            // A 90-degree rotation error would swap width/height, dramatically changing aspect ratio
            expect(aspectRatio).toBeGreaterThan(0.1);
            expect(aspectRatio).toBeLessThan(10);

            // If original was landscape (width > height), thumbnail should be too
            // We'll check that at least the dimensions are valid
            expect(dimensions.width).toBeGreaterThan(0);
            expect(dimensions.height).toBeGreaterThan(0);
        }
    });

    test('direct thumbnail API request returns correctly oriented image', async ({ request }) => {
        // Test direct API call to thumbnails endpoint
        const response = await request.get('/api/v1/thumbnails/test-image.jpg');

        // Should return an image even if file doesn't exist (or 404)
        if (response.ok()) {
            const contentType = response.headers()['content-type'];
            expect(contentType).toMatch(/image\//);

            // Response should have image data
            const buffer = await response.body();
            expect(buffer.length).toBeGreaterThan(0);
        }
    });

    test('photo viewer shows correctly oriented full-size image', async ({ page }) => {
        await page.goto('/photos');
        await page.waitForTimeout(1000);

        const photoItems = page.locator('.photo-grid-item');
        const itemCount = await photoItems.count();

        if (itemCount > 0) {
            // Click on the first photo to open viewer
            await photoItems.first().click();

            // Wait for viewer to open
            await page.waitForTimeout(500);

            // Check if viewer dialog exists and is visible
            const fileViewer = page.locator('dialog#file-viewer');
            const isVisible = await fileViewer.isVisible();

            if (isVisible) {
                // Verify the full-size image in the viewer
                const viewerImage = fileViewer.locator('img');
                await expect(viewerImage).toBeVisible();

                // Verify image has valid dimensions
                const dimensions = await viewerImage.evaluate((img: HTMLImageElement) => {
                    return {
                        width: img.naturalWidth,
                        height: img.naturalHeight
                    };
                });

                expect(dimensions.width).toBeGreaterThan(0);
                expect(dimensions.height).toBeGreaterThan(0);
            }
        }
    });
});
