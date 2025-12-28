<template>
  <main class="site-main-bg">
    <TopNav
      :nav-links="navLinks"
      :is-minimal="isMinimal"
      :minimize-key-combo="minimizeKeyCombo"
    />
    <div
      style="
        position: fixed;
        top: 1rem;
        right: 1rem;
        z-index: 9999;
        background: #fff3;
        padding: 0.5rem 1rem;
        border-radius: 8px;
        backdrop-filter: blur(4px);
      "
    >
      <label style="font-size: 0.9em"
        >Font size scale:
        <input
          type="range"
          min="0.8"
          max="1.5"
          step="0.01"
          :value="theme.fontSizeScale"
          @input="
            theme.setFontSizeScale(
              Number(($event.target as HTMLInputElement).value),
            )
          "
          style="vertical-align: middle; width: 100px"
        />
        <span>{{ theme.fontSizeScale.toFixed(2) }}x</span>
      </label>
    </div>
    <RouterView />
  </main>
</template>

<script lang="ts" setup>
import TopNav from '@/components/home/TopNav.vue'
import { useThemeStore } from '@/stores/theme'
import type { NavLink } from '@/types/nav_link'
import { onMounted, ref } from 'vue'
import { RouterView } from 'vue-router'
import { fromKeyComboString, toEventListenerFunc } from './util/keycombo'

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
  font-size: $font-size-xs;
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
    font-size: $font-size-base;

    padding: 1rem 0;
    margin-top: 1rem;
  }
}
</style>
