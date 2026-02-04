<template>
  <LibraryLayout>
    <template #sidebar>
      <BooksSidebar />
    </template>
    <template #title>
      <span class="mock-badge">mock</span>
      <h2 class="library-title">Library</h2>
    </template>
    <template #subtitle>
      <div class="library-subtitle">
        {{ formatBookCount(totalBooks) }}
      </div>
    </template>
    <template #main>
      <div id="books-view">
        <div v-if="books.length === 0" class="books-empty">
          <h2>No books found</h2>
          <p>Add PDF or EPUB files to your files directory to see them here.</p>
        </div>
        <div v-else class="books-grid">
          <div
            v-for="book in books"
            :key="book.relPath"
            @click.prevent="selectBook(book)"
          >
            <div class="book-card">
              <div class="book-card-cover">
                <span class="book-card-badge">{{ book.type }}</span>
              </div>
              <div class="book-card-info">
                <h3 class="book-card-title" :title="book.fileName">
                  {{ book.title }}
                </h3>
                <p class="book-card-size">{{ formatBookSize(book.size) }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
      <CirrusFileViewer
        v-model="fileViewerOpen"
        :file-src="selectedFileSrc"
        :file-type="selectedFileType"
      />
    </template>
  </LibraryLayout>
</template>

<script lang="ts" setup>
import BooksSidebar from '@/components/books/BooksSidebar.vue';
import CirrusFileViewer from '@/components/cirrus/CirrusFileViewer.vue';
import LibraryLayout from '@/components/common/LibraryLayout.vue';
import BooksService, { type BookApiResponse } from '@/services/booksService';
import CirrusService from '@/services/cirrusService';
import type { Book } from '@/types/book';
import type { FileType } from '@/types/cirrus';
import { onMounted, ref } from 'vue';

const books = ref<Book[]>([]);
const totalBooks = ref(0);

const fileViewerOpen = ref(false);
const selectedFileSrc = ref('');
const selectedFileType = ref<FileType>('pdf');

const selectBook = (book: Book) => {
  console.log('Selected book:', book);
  if (book.relPath) {
    selectedFileSrc.value = CirrusService.getDownloadUrl(book.relPath);
    selectedFileType.value = book.type.toLowerCase() as FileType;
    fileViewerOpen.value = true;
  }
};

const formatBookCount = (count: number) => {
  if (count === 1) return '1 book';
  return `${count.toLocaleString()} books`;
};

const formatBookSize = (size: number) => {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`;
  return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`;
};

const cleanBookTitle = (fileName: string): string =>
  fileName
    .replace(/\.[^.]+$/, '')
    .replace(/[_-]+/g, ' ')
    .trim();

const convertBookApi = (book: BookApiResponse) => ({
  relPath: book.relPath,
  fileName: book.fileName,
  title: cleanBookTitle(book.fileName),
  size: book.size,
  type: book.type.toUpperCase(),
});

onMounted(async () => {
  try {
    const bookList = await BooksService.listBooks();
    books.value = bookList.map(convertBookApi);
    totalBooks.value = bookList.length;
  } catch (e) {
    // TODO: handle error
    console.error(e);
  }
});
</script>

<style lang="scss" scoped>
.book-card {
  cursor: pointer;
  border-radius: $border-radius;
  box-shadow: $shadow-sm;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: box-shadow 0.2s;
  border: 1px solid $theme-palette-border;
  background: $theme-palette-bg-primary;

  @media (prefers-color-scheme: dark) {
    background: $theme-palette-bg-nav;
  }

  &:hover {
    box-shadow: $shadow-md;
    border-color: $theme-palette-accent;
  }
}

.book-card-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  background: $theme-palette-accent;
  color: $theme-palette-text-inverse;
  font-size: $theme-font-size-xs;
  padding: 2px 8px;
  border-radius: 8px;
}

.book-card-cover {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 120px;
  background: $theme-palette-bg-secondary;
  position: relative;

  @media (prefers-color-scheme: dark) {
    background: $theme-palette-bg-nav;
  }
}

.book-card-info {
  padding: $spacing-md;
}

.book-card-size {
  font-size: $theme-font-size-xs;
  color: $theme-palette-text-muted;
}

.book-card-title {
  font-size: $theme-font-size-base;
  font-weight: 600;
  margin: 0 0 4px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: $theme-palette-text-primary;

  @media (prefers-color-scheme: dark) {
    color: $theme-palette-text-primary;
  }
}

.books-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  padding: $spacing-2xl;
  color: $theme-palette-text-muted;
}

.books-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: $spacing-xl;
  width: 100vw;
  max-width: 1400px;
  margin: 0 auto;
  padding-bottom: $spacing-2xl;
}

.books-header-simple {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: $spacing-2xl;
  margin-bottom: $spacing-xl;
}

.books-library-count {
  font-size: $theme-font-size-lg;
  color: $theme-palette-text-muted;
}

.books-library-title {
  font-size: $theme-font-size-2xl;
  font-weight: 700;
  margin: 0;
  display: flex;
  align-items: center;
  gap: $spacing-md;
  color: $theme-palette-text-primary;

  @media (prefers-color-scheme: dark) {
    color: $theme-palette-text-primary;
  }
}

.mock-badge {
  background: $theme-palette-bg-secondary;
  color: $theme-palette-text-secondary;
  font-size: $theme-font-size-xs;
  padding: 2px 8px;
  border-radius: 8px;
  text-transform: uppercase;
  font-weight: 700;
}
</style>
