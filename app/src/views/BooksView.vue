<template>
  <div class="landing-body">
    <TopNav :navLinks="navLinks" />
    <div class="books-header-simple">
      <h1 class="books-library-title">
        Library <span class="mock-badge">mock</span>
      </h1>
      <p class="books-library-count">{{ formatBookCount(totalBooks) }}</p>
    </div>
    <div v-if="books.length === 0" class="books-empty">
      <div class="book-card-icon">
        <svg width="32" height="32" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6l4 2"/></svg>
      </div>
      <h2>No books found</h2>
      <p>Add PDF or EPUB files to your files directory to see them here.</p>
    </div>
    <div v-else class="books-grid-simple">
      <router-link v-for="book in books" :key="book.relPath" :to="`/books/reader?path=${encodeURIComponent(book.relPath)}`" class="book-card-link">
        <div class="book-card">
          <div class="book-card-cover">
            <div class="book-card-icon">
              <svg width="32" height="32" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6l4 2"/></svg>
            </div>
            <span class="book-card-badge">{{ book.type }}</span>
          </div>
          <div class="book-card-info">
            <h3 class="book-card-title" :title="book.fileName">{{ book.title }}</h3>
            <p class="book-card-size">{{ formatBookSize(book.size) }}</p>
          </div>
        </div>
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import TopNav from '@/components/home/TopNav.vue'
import type { NavLink } from '@/types/home'

import { ref, onMounted } from 'vue'
import { fetchBooks, type BookApiResponse } from '@/services/booksService'

const navLinks: NavLink[] = [
  { name: 'Cirrus', href: '/cirrus' },
  { name: 'Photos', href: '/photos' },
  { name: 'Books', href: '/books' },
]

interface Book {
  relPath: string
  fileName: string
  title: string
  size: number
  type: string
}

const books = ref<Book[]>([])
const totalBooks = ref(0)

function formatBookCount(count: number) {
  if (count === 1) return '1 book'
  return `${count.toLocaleString()} books`
}

function formatBookSize(size: number) {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`
}

function cleanBookTitle(fileName: string): string {
  // Remove extension and replace underscores/dashes with spaces
  return fileName.replace(/\.[^.]+$/, '').replace(/[_-]+/g, ' ').trim()
}

function convertBookApi(book: BookApiResponse) {
  return {
    relPath: book.relPath,
    fileName: book.fileName,
    title: cleanBookTitle(book.fileName),
    size: book.size,
    type: book.type.toUpperCase(),
  }
}

onMounted(async () => {
  try {
    const bookList = await fetchBooks()
    books.value = bookList.map(convertBookApi)
    totalBooks.value = bookList.length
  } catch (e) {
    // TODO: handle error
    console.error(e)
  }
})
</script>

<style scoped>
.books-header-simple {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: var(--spacing-2xl);
  margin-bottom: var(--spacing-xl);
}
.books-library-title {
  font-size: var(--font-size-2xl);
  font-weight: 700;
  color: var(--color-gray-900);
  margin: 0;
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
}

@media (prefers-color-scheme: dark) {
  .books-library-title {
    color: white;
  }
}
.books-library-count {
  font-size: var(--font-size-lg);
  color: var(--color-gray-500);
}
.books-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  padding: var(--spacing-2xl);
  color: var(--color-gray-500);
}
.books-grid-simple {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--spacing-xl);
  width: 100vw;
  max-width: 1400px;
  margin: 0 auto;
  padding-bottom: var(--spacing-2xl);
}
/* Book card styling similar to photo grid */
.book-card {
  background: var(--color-gray-900);
  border-radius: var(--border-radius);
  box-shadow: var(--shadow-sm);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: box-shadow 0.2s;
  border: 1px solid var(--color-gray-800);
}
.book-card:hover {
  box-shadow: var(--shadow-md);
  border-color: var(--color-primary-400);
}
.book-card-cover {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 120px;
  background: var(--color-gray-800);
  position: relative;
}
.book-card-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  background: var(--color-primary-600);
  color: white;
  font-size: var(--font-size-xs);
  padding: 2px 8px;
  border-radius: 8px;
}
.book-card-info {
  padding: var(--spacing-md);
}
.book-card-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: white;
  margin: 0 0 4px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.book-card-size {
  font-size: var(--font-size-xs);
  color: var(--color-gray-300);
}
.book-card-link {
  text-decoration: none;
  color: inherit;
}
.mock-badge {
  margin-left: 1rem;
  background: var(--color-gray-300);
  color: var(--color-gray-700);
  font-size: var(--font-size-xs);
  padding: 2px 8px;
  border-radius: 8px;
  text-transform: uppercase;
  font-weight: 700;
}
</style>
