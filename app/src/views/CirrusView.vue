<script setup lang="ts">
import TopNav from '@/components/home/TopNav.vue'
import CirrusFileExplorer from '@/components/cirrus/CirrusFileExplorer.vue'
import type { NavLink } from '@/types/home'

const navLinks: NavLink[] = [
  { name: 'Cirrus', href: '/cirrus' },
  { name: 'Photos', href: '/photos' },
  { name: 'Books', href: '/books' },
]
</script>

<template>
  <main class="cirrus-view">
    <TopNav :navLinks="navLinks" />
    <CirrusFileExplorer />
  </main>
</template>

<style lang="scss" scoped>
.cirrus-view {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: linear-gradient(180deg, hsl(225, 25%, 15%) 0%, hsl(225, 30%, 10%) 100%);

  @media (prefers-color-scheme: light) {
    background: linear-gradient(180deg, hsl(225, 15%, 95%) 0%, hsl(225, 20%, 98%) 100%);
  }
}
</style>
