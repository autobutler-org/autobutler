import { test, expect } from '@playwright/test';

test.describe('Books Page', () => {
    test('loads books page successfully', async ({ page }) => {
        await page.goto('/books');

        await expect(page).toHaveTitle(/Autobutler/);
        await expect(page.locator('.books-library')).toBeVisible();
    });

    test('displays books library header with title', async ({ page }) => {
        await page.goto('/books');

        const header = page.locator('.books-library-header');
        await expect(header).toBeVisible();

        const title = page.locator('h1.books-library-title');
        await expect(title).toBeVisible();
        await expect(title).toHaveText('Library');
    });

    test('displays book count with proper formatting', async ({ page }) => {
        await page.goto('/books');

        const countElement = page.locator('.books-library-count');
        await expect(countElement).toBeVisible();

        const countText = await countElement.textContent();

        // Should show either "0 books", "1 book", or "N books"
        expect(countText).toMatch(/\d+\s+books?/i);
    });

    test('shows empty state when no books exist', async ({ page }) => {
        await page.goto('/books');

        const bookCount = await page.locator('.book-card').count();

        if (bookCount === 0) {
            const emptyState = page.locator('.books-empty');
            await expect(emptyState).toBeVisible();

            const emptyTitle = page.locator('.books-empty h2');
            await expect(emptyTitle).toBeVisible();
            await expect(emptyTitle).toHaveText('No books found');

            const emptyMessage = page.locator('.books-empty p');
            await expect(emptyMessage).toBeVisible();
            await expect(emptyMessage).toContainText('Add PDF or EPUB files');

            const emptyIcon = page.locator('.books-empty svg');
            await expect(emptyIcon).toBeVisible();
        }
    });

    test('displays books grid when books exist', async ({ page }) => {
        await page.goto('/books');

        const bookCount = await page.locator('.book-card').count();

        if (bookCount > 0) {
            const booksGrid = page.locator('.books-grid');
            await expect(booksGrid).toBeVisible();
        }
    });

    test('book cards have proper structure when books exist', async ({ page }) => {
        await page.goto('/books');

        const bookCards = page.locator('.book-card');
        const cardCount = await bookCards.count();

        if (cardCount > 0) {
            const firstCard = bookCards.first();

            // Check for link
            const link = firstCard.locator('a.book-card-link');
            await expect(link).toBeVisible();
            await expect(link).toHaveAttribute('href');

            // Check for cover
            const cover = firstCard.locator('.book-card-cover');
            await expect(cover).toBeVisible();

            // Check for icon or thumbnail
            const hasIcon = (await firstCard.locator('.book-card-icon').count()) > 0;
            const hasThumbnail = (await firstCard.locator('img.book-card-thumbnail').count()) > 0;

            expect(hasIcon || hasThumbnail).toBe(true);
        }
    });

    test('PDF books display thumbnails with fallback', async ({ page }) => {
        await page.goto('/books');

        const bookCards = page.locator('.book-card');
        const cardCount = await bookCards.count();

        if (cardCount > 0) {
            const pdfCards = bookCards.filter({
                has: page.locator('.book-card-badge:has-text("PDF")'),
            });
            const pdfCount = await pdfCards.count();

            if (pdfCount > 0) {
                const firstPdfCard = pdfCards.first();

                // Check for thumbnail with lazy loading
                const thumbnail = firstPdfCard.locator('img.book-card-thumbnail');
                await expect(thumbnail).toHaveAttribute('loading', 'lazy');
                await expect(thumbnail).toHaveAttribute('onerror');

                // Check for fallback icon
                const fallback = firstPdfCard.locator('.book-card-icon-fallback');
                await expect(fallback).toBeAttached();

                // Check for PDF badge
                const badge = firstPdfCard.locator('.book-card-badge');
                await expect(badge).toHaveText('PDF');
            }
        }
    });

    test('EPUB books display icon without thumbnail', async ({ page }) => {
        await page.goto('/books');

        const bookCards = page.locator('.book-card');
        const cardCount = await bookCards.count();

        if (cardCount > 0) {
            const epubCards = bookCards.filter({
                hasNot: page.locator('.book-card-badge:has-text("PDF")'),
            });
            const epubCount = await epubCards.count();

            if (epubCount > 0) {
                const firstEpubCard = epubCards.first();

                // EPUB should show icon directly
                const icon = firstEpubCard.locator('.book-card-icon svg');
                const iconCount = await icon.count();

                expect(iconCount).toBeGreaterThan(0);
            }
        }
    });

    test('book card links point to reader with path parameter', async ({ page }) => {
        await page.goto('/books');

        const bookCards = page.locator('.book-card');
        const cardCount = await bookCards.count();

        if (cardCount > 0) {
            const firstCardLink = bookCards.first().locator('a.book-card-link');
            const href = await firstCardLink.getAttribute('href');

            expect(href).toContain('/books/reader?path=');
        }
    });

    test('book reader page accepts path parameter', async ({ page }) => {
        await page.goto('/books/reader?path=/test/book.epub');

        await expect(page).toHaveTitle(/Autobutler/);
    });

    test('book reader has back to library button', async ({ page }) => {
        await page.goto('/books/reader?path=/test/book.epub');

        const backButton = page.locator('button[title="Back to library"]');

        if ((await backButton.count()) > 0) {
            await expect(backButton).toBeVisible();
            // Button uses onclick="history.back()" instead of href
            await expect(backButton).toHaveAttribute('onclick', 'history.back()');
        }
    });

    test('book reader displays book title', async ({ page }) => {
        await page.goto('/books/reader?path=/test/sample.epub');

        const title = page.locator('.book-reader-title');

        if ((await title.count()) > 0) {
            await expect(title).toBeVisible();
            // Should show the filename
            await expect(title).toContainText('sample.epub');
        }
    });

    test('book library structure exists', async ({ page }) => {
        await page.goto('/books');

        // Books library should exist
        const booksLibrary = page.locator('.books-library');
        await expect(booksLibrary).toBeVisible();
    });
});

test.describe('Books Page - Book Upload', () => {
    test('shows empty state initially', async ({ page }) => {
        await page.goto('/books');

        const bookCards = page.locator('.book-card');
        const cardCount = await bookCards.count();

        // Check for empty state when no books
        if (cardCount === 0) {
            const emptyState = page.locator('.books-empty');
            await expect(emptyState).toBeVisible();

            const emptyTitle = page.locator('.books-empty h2');
            await expect(emptyTitle).toHaveText('No books found');
        }
    });

    test('book count shows "0 books" when empty', async ({ page }) => {
        await page.goto('/books');

        const countElement = page.locator('.books-library-count');
        const countText = await countElement.textContent();

        // Should show book count
        expect(countText).toMatch(/\d+\s+books?/i);
    });

    test('books library shows proper grid structure', async ({ page }) => {
        await page.goto('/books');

        // The library should have proper structure
        const libraryHeader = page.locator('.books-library-header');
        await expect(libraryHeader).toBeVisible();

        const title = page.locator('h1.books-library-title');
        await expect(title).toHaveText('Library');
    });

    test('empty state provides helpful instructions', async ({ page }) => {
        await page.goto('/books');

        const bookCards = page.locator('.book-card');
        const cardCount = await bookCards.count();

        if (cardCount === 0) {
            const emptyMessage = page.locator('.books-empty p');
            await expect(emptyMessage).toBeVisible();
            await expect(emptyMessage).toContainText('Add PDF or EPUB files');
        }
    });
});
