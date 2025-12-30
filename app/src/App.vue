<template>
  <main class="site-main-bg">
    <TopNav
      :nav-links="navLinks"
      :is-minimal="isMinimal"
      :minimize-key-combo="minimizeKeyCombo"
    />
    <ThemeModal :open="themeModalOpen" @close="themeModalOpen = false" />
    <RouterView />
  </main>
</template>

<script lang="ts" setup>
import TopNav from '@/components/home/TopNav.vue'
import ThemeModal from '@/components/ThemeModal.vue'
import { useThemeStore } from '@/stores/theme'
import type { NavLink } from '@/types/nav_link'
import { fromKeyComboString, toEventListenerFunc } from '@/util/keycombo'
import { onBeforeUnmount, onMounted, ref, ref as vueRef } from 'vue'
import { RouterView } from 'vue-router'

const navLinks: NavLink[] = [
  { name: 'Cirrus', href: '/cirrus' },
  { name: 'Photos', href: '/photos' },
  { name: 'Books', href: '/books' },
]

const isMinimal = ref(false)
const toggleMinimize = (_: Event): boolean => {
  isMinimal.value = !isMinimal.value
  return true
}
const minimizeKeyCombo = fromKeyComboString('alt-space')
const theme = useThemeStore()
const themeModalOpen = vueRef(false)

const handleSettingsHotkey = (e: KeyboardEvent) => {
  if (e.altKey && (e.key === '`' || e.code === 'Backquote')) {
    e.preventDefault()
    themeModalOpen.value = !themeModalOpen.value
  }
}
onMounted(() => {
  window.addEventListener('keydown', handleSettingsHotkey)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleSettingsHotkey)
})
onMounted(() => {
  theme.applyFontSizeScale()
})
document.addEventListener(
  'keydown',
  toEventListenerFunc(minimizeKeyCombo, toggleMinimize),
)
</script>

<style lang="scss" scoped>
* {
  color: $theme-palette-text-primary;
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
  font-size: $theme-font-size-xs;
  text-align: center;
  margin-top: 2rem;

  a {
    display: inline-block;
    padding: 0 1rem;
    border-left: 1px solid $theme-palette-border;

    &.router-link-exact-active {
      color: $theme-palette-text-secondary;

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
    font-size: $theme-font-size-base;

    padding: 1rem 0;
    margin-top: 1rem;
  }
}
</style>
