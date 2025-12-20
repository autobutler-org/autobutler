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
  <main class="site-main-bg">
    <TopNav :navLinks="navLinks" />
    <CirrusFileExplorer />
  </main>
</template>

<style lang="scss" scoped>

</style>
