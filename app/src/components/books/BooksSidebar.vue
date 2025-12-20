<template>
  <nav class="books-sidebar">
    <div class="books-sidebar-section">
      <h2 class="books-sidebar-title">Library</h2>
      <ul class="books-sidebar-list">
        <li class="books-sidebar-item active">All Books</li>
        <li class="books-sidebar-item">Shelves</li>
        <li class="books-sidebar-item">Favorites</li>
        <li class="books-sidebar-item">Recently Added</li>
      </ul>
    </div>
    <div class="books-sidebar-section">
      <h2 class="books-sidebar-title">Smart Shelves</h2>
      <ul class="books-sidebar-list">
        <li class="books-sidebar-item">PDFs</li>
        <li class="books-sidebar-item">EPUBs</li>
      </ul>
    </div>
  </nav>
</template>

<script setup lang="ts">
// Stub: No props or logic yet
</script>

<style scoped>
.books-sidebar {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2xl);
  padding: var(--spacing-xl) var(--spacing-lg);
  width: 100%;
  box-sizing: border-box;
}
.books-sidebar-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}
.books-sidebar-title {
  font-size: var(--font-size-sm);
  font-weight: 700;
  color: var(--color-gray-400);
  margin-bottom: var(--spacing-xs);
  text-transform: uppercase;
  letter-spacing: 1px;
}
.books-sidebar-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}
.books-sidebar-item {
  font-size: var(--font-size-base);
  color: var(--color-gray-300);
  padding: 4px 0;
  cursor: pointer;
  border-radius: 6px;
  transition: background 0.2s, color 0.2s;
}
.books-sidebar-item.active {
  color: var(--color-primary-500);
  background: var(--color-primary-50);
}
.books-sidebar-item:hover {
  background: var(--color-gray-800);
  color: var(--color-primary-400);
}
</style>
