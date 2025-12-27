<template>
  <main class="site-main-bg">
    <TopNav :navLinks="navLinks" :is-minimal="isMinimal" />
    <RouterView />
  </main>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router'
import TopNav from '@/components/home/TopNav.vue'
import type { NavLink } from '@/types/nav_link'
import { ref } from 'vue'

const navLinks: NavLink[] = [
  { name: 'Cirrus', href: '/cirrus' },
  { name: 'Photos', href: '/photos' },
  { name: 'Books', href: '/books' },
]

const isMinimal = ref(false)

const toggleMinimize = () => {
  isMinimal.value = !isMinimal.value;
}

window.addEventListener('keydown', (e) => {
  if (e.altKey && e.code === 'Space') {
    toggleMinimize()
  }
})
</script>

<style lang="scss" scoped>
* {
  color: white;
}

header {
  line-height: 1.5;
  max-height: 100vh;

  @media (min-width: 1024px) {
    display: flex;
    place-items: center;
    padding-right: calc($section-gap) / 2;
  }

  .wrapper {
    @media (min-width: 1024px) {
      display: flex;
      place-items: flex-start;
      flex-wrap: wrap;
    }
  }
}

.logo {
  display: block;
  margin: 0 auto 2rem;

  @media (min-width: 1024px) {
    margin: 0 2rem 0 0;
  }
}

nav {
  width: 100%;
  font-size: 12px;
  text-align: center;
  margin-top: 2rem;

  a {
    display: inline-block;
    padding: 0 1rem;
    border-left: 1px solid $color-border;

    &.router-link-exact-active {
      color: $color-text-secondary;

      &:hover {
        background-color: transparent;
      }
    }

    &:first-of-type {
      border: 0;
    }
  }

  @media (min-width: 1024px) {
    text-align: left;
    margin-left: -1rem;
    font-size: 1rem;

    padding: 1rem 0;
    margin-top: 1rem;
  }
}
</style>
