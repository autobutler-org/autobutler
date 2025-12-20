<template>
  <div class="landing-body">
    <TopNav :navLinks="navLinks" />
    <div class="library-layout">
      <div class="library-container">
        <aside class="library-sidebar">
          <slot name="sidebar" />
        </aside>
        <div class="library-separator" aria-hidden="true"></div>
        <main class="library-main">
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
  background: transparent;
  padding: 0;
}
</style>
