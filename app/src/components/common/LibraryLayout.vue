<template>
  <div class="landing-body">
    <TopNav :navLinks="navLinks" />
    <div class="photos-library">
      <div class="photos-container">
        <aside class="photos-sidebar">
          <slot name="sidebar" />
        </aside>
        <div class="photos-separator" aria-hidden="true"></div>
        <main class="photos-main">
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
.photos-library {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100vw;
}
.photos-container {
  display: flex;
  height: 100vh;
  max-height: 100vh;
  max-width: 100vw;
  overflow: hidden;
}
.photos-sidebar {
  width: 280px;
  min-width: 220px;
  max-width: 320px;
  background: transparent;
  padding: 0;
  display: flex;
  flex-direction: column;
}
.photos-separator {
  width: 1px;
  height: 100vh;
  background: var(--color-gray-900);
  opacity: 0.12;
  border-radius: 2px;
  align-self: stretch;
  transition: background 0.2s;
}
.photos-main {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: transparent;
  padding: 0;
}
</style>
