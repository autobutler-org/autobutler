<template>
  <div class="photos-sidebar">
    <!-- Library Section -->
    <div class="sidebar-section">
      <h3 class="sidebar-title">Library</h3>
      <nav class="sidebar-nav">
        <a href="/photos" class="sidebar-link sidebar-link--active">
          <svg xmlns="http://www.w3.org/2000/svg" class="sidebar-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <span>All Photos</span>
          <span class="sidebar-count">12,842</span>
        </a>
        <a href="#" class="sidebar-link">
          <svg xmlns="http://www.w3.org/2000/svg" class="sidebar-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <span>Albums</span>
          <span class="sidebar-count">32</span>
        </a>
        <a href="#" class="sidebar-link">
          <svg xmlns="http://www.w3.org/2000/svg" class="sidebar-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
          </svg>
          <span>Favorites</span>
          <span class="sidebar-count">418</span>
        </a>
        <a href="#" class="sidebar-link">
          <svg xmlns="http://www.w3.org/2000/svg" class="sidebar-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>Recently Added</span>
          <span class="sidebar-count">296</span>
        </a>
      </nav>
    </div>
    <div class="sidebar-section">
      <h3 class="sidebar-title">Smart Albums</h3>
      <nav class="sidebar-nav">
        <a href="#" class="sidebar-link">
          <svg xmlns="http://www.w3.org/2000/svg" class="sidebar-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
          <span>Screenshots</span>
          <span class="sidebar-count">1,204</span>
        </a>
        <a href="#" class="sidebar-link">
          <svg xmlns="http://www.w3.org/2000/svg" class="sidebar-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
          <span>Videos</span>
          <span class="sidebar-count">643</span>
        </a>
        <a href="#" class="sidebar-link">
          <svg xmlns="http://www.w3.org/2000/svg" class="sidebar-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>Selfies</span>
          <span class="sidebar-count">312</span>
        </a>
      </nav>
    </div>
    <div class="sidebar-section sidebar-storage">
      <!-- Storage bar placeholder -->
    </div>
  </div>
</template>

<script setup lang="ts">
// TODO: Accept summary prop and render storage bar
</script>

<style lang="scss" scoped>
.photos-sidebar {
  width: 250px;
  background: var(--color-gray-900);
  color: white;
  padding: var(--spacing-xl);
  overflow-y: auto;
  flex-shrink: 0;
  border-right: 1px solid var(--color-gray-800);
}
.sidebar-section {
  margin-bottom: var(--spacing-2xl);
}
.sidebar-title {
  font-size: var(--font-size-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-gray-400);
  margin-bottom: var(--spacing-md);
}
.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}
.sidebar-link {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-md);
  border-radius: var(--border-radius);
  color: var(--color-gray-300);
  text-decoration: none;
  transition: all 0.2s ease;
  font-size: var(--font-size-sm);
}
.sidebar-link:hover {
  background: var(--color-gray-800);
  color: white;
}
.sidebar-link--active {
  background: var(--color-primary-600);
  color: white;
}
.sidebar-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.sidebar-count {
  margin-left: auto;
  font-size: var(--font-size-xs);
  color: var(--color-gray-500);
}
.sidebar-link--active .sidebar-count {
  color: var(--color-primary-100);
}
.sidebar-badge {
  margin-left: auto;
  font-size: var(--font-size-xs);
  background: var(--color-gray-800);
  padding: 2px 8px;
  border-radius: 10px;
  color: var(--color-gray-400);
}
.sidebar-storage {
  margin-top: auto;
  padding-top: var(--spacing-xl);
  border-top: 1px solid var(--color-gray-800);
}
</style>
