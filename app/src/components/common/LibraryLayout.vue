<template>
  <div class="site-main-bg">
    <TopNav :navLinks="navLinks" />
    <div class="library-layout">
      <div class="library-container">
        <aside class="library-sidebar">
          <slot name="sidebar" />
        </aside>
        <div class="library-separator" aria-hidden="true"></div>
        <main class="library-main">
          <div class="library-header">
            <slot name="header-title" />
            <slot name="header-subtitle" />
          </div>
          <slot name="main" />
        </main>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import TopNav from '@/components/home/TopNav.vue'
import type { NavLink } from '@/types/home'

const navLinks: NavLink[] = [
  { name: 'Cirrus', href: '/cirrus' },
  { name: 'Photos', href: '/photos' },
  { name: 'Books', href: '/books' },
]
</script>

<style scoped>
/* Generic library layout classes */
.library-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-lg) 0;
}
.library-layout {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100vw;
}
.library-container {
  display: flex;
  height: 100vh;
  max-height: 100vh;
  max-width: 100vw;
  overflow: hidden;
}
.library-sidebar {
  width: 280px;
  min-width: 220px;
  max-width: 320px;
  background: transparent;
  padding: 0;
  display: flex;
  flex-direction: column;
}
.library-separator {
  width: 1px;
  height: 100vh;
  background: var(--color-gray-900);
  align-self: stretch;
}
.library-main {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: var(--color-background-soft);
  padding: 0;
}
@media (prefers-color-scheme: dark) {
  .library-main {
    background: var(--color-background-mute);
  }
}
.library-title {
  font-size: var(--font-size-3xl);
  font-weight: 700;
  margin: 0;
  color: var(--color-gray-700);
}
@media (prefers-color-scheme: dark) {
  .library-title {
    color: var(--color-gray-100);
  }
}
.library-subtitle {
  font-size: var(--font-size-lg);
  color: var(--color-gray-500);
}
</style>
