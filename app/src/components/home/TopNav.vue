<template>
  <nav class="landing-nav">
    <div class="landing-nav-left">
      <RouterLink to="/" class="landing-nav-logo">
        <img src="/img/butler.png" alt="AutoButler" />
      </RouterLink>
      <RouterLink
        v-for="link in navLinks"
        :key="link.href"
        :to="link.href"
        class="landing-nav-link"
      >
        {{ link.name }}
      </RouterLink>
    </div>
    <div class="landing-nav-right">
      <RouterLink to="/settings" class="landing-nav-button" title="Settings">
        <SettingsIcon />
      </RouterLink>
      <div class="version-container" id="version-container">
        <button
          class="landing-nav-version version-display"
          :title="'Version ' + currentVersion"
          @click.stop="toggleVersionDropdown"
        >
          {{ currentVersion }}
          <span style="margin-left: 0.25rem">▾</span>
        </button>
        <div v-if="versionDropdownOpen" class="version-dropdown" @click.stop>
          <div v-if="loadingReleases" class="version-dropdown-loading">Loading...</div>
          <template v-else-if="releases.length > 0">
            <a
              v-for="release in releases"
              :key="release.tagName"
              :href="release.htmlUrl"
              target="_blank"
              rel="noopener noreferrer"
              :class="[
                'version-dropdown-item',
                { 'version-dropdown-item--current': release.isCurrentVersion },
              ]"
            >
              <span class="version-dropdown-tag">{{ release.tagName }}</span>
              <span v-if="release.isCurrentVersion" class="version-dropdown-badge">Current</span>
            </a>
          </template>
          <div v-else class="version-dropdown-empty">No releases available</div>
        </div>
      </div>
      <RouterLink to="/devices" class="landing-nav-button" title="Devices">
        <DeviceIcon />
        <span>Devices</span>
      </RouterLink>
      <button class="landing-nav-hamburger" @click="toggleMobileMenu" aria-label="Menu">
        <HamburgerIcon />
      </button>
    </div>
  </nav>

  <!-- Mobile Menu -->
  <div
    id="mobile-menu"
    :class="['mobile-menu', { 'mobile-menu--open': mobileMenuOpen }]"
    @click.self="closeMobileMenu"
  >
    <div class="mobile-menu-content">
      <div class="mobile-menu-header">
        <span class="mobile-menu-title">Menu</span>
        <button class="mobile-menu-close" @click="closeMobileMenu" aria-label="Close menu">
          <CloseIcon />
        </button>
      </div>
      <nav class="mobile-menu-nav">
        <RouterLink to="/" class="mobile-menu-link" @click="closeMobileMenu">
          <HomeIcon />
          <span>Home</span>
        </RouterLink>
        <RouterLink to="/cirrus" class="mobile-menu-link" @click="closeMobileMenu">
          <FolderIcon />
          <span>Cirrus</span>
        </RouterLink>
        <RouterLink to="/photos" class="mobile-menu-link" @click="closeMobileMenu">
          <PhotoIcon />
          <span>Photos</span>
        </RouterLink>
        <RouterLink to="/books" class="mobile-menu-link" @click="closeMobileMenu">
          <BookIcon />
          <span>Books</span>
        </RouterLink>
        <RouterLink to="/devices" class="mobile-menu-link" @click="closeMobileMenu">
          <DeviceIcon />
          <span>Devices</span>
        </RouterLink>
      </nav>
      <div class="mobile-menu-footer">
        <RouterLink
          to="/settings"
          class="mobile-menu-link"
          title="Settings"
          @click="closeMobileMenu"
        >
          <SettingsIcon />
          <span>Settings</span>
        </RouterLink>
        <div class="mobile-menu-divider"></div>
        <span class="mobile-menu-version">Version: {{ currentVersion }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import type { NavLink } from '@/types/nav_link'
import { RouterLink } from 'vue-router'
import { getCurrentVersion, getAvailableReleases, type Release } from '@/services/versionService'
import SettingsIcon from '../icons/SettingsIcon.vue'
import DeviceIcon from '../icons/DeviceIcon.vue'
import HamburgerIcon from '../icons/HamburgerIcon.vue'
import CloseIcon from '../icons/CloseIcon.vue'
import HomeIcon from '../icons/HomeIcon.vue'
import FolderIcon from '../icons/FolderIcon.vue'
import PhotoIcon from '../icons/PhotoIcon.vue'
import BookIcon from '../icons/BookIcon.vue'

defineProps<{
  navLinks?: NavLink[]
}>()

const mobileMenuOpen = ref(false)
const versionDropdownOpen = ref(false)
const currentVersion = ref('vX.Y.Z')
const releases = ref<Release[]>([])
const loadingReleases = ref(false)

// Fetch current version on mount
onMounted(async () => {
  currentVersion.value = await getCurrentVersion()

  // Add click outside listener for version dropdown
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

const handleClickOutside = (event: MouseEvent) => {
  const container = document.getElementById('version-container')
  if (container && !container.contains(event.target as Node)) {
    versionDropdownOpen.value = false
  }
}

const toggleVersionDropdown = async () => {
  if (versionDropdownOpen.value) {
    versionDropdownOpen.value = false
    return
  }

  versionDropdownOpen.value = true
  if (releases.value.length === 0) {
    loadingReleases.value = true
    releases.value = await getAvailableReleases()
    loadingReleases.value = false
  }
}

const toggleMobileMenu = () => {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

const closeMobileMenu = () => {
  mobileMenuOpen.value = false
}
</script>

<style lang="scss" scoped>
.landing-nav {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-lg $spacing-2xl;
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  width: 100vw;
  margin-left: calc(50% - 50vw);
  margin-right: calc(50% - 50vw);
  left: 0;
  right: 0;

  @media (prefers-color-scheme: light) {
    background: rgba(255, 255, 255, 0.8);
    border-bottom-color: rgba(0, 0, 0, 0.1);
  }
}

.landing-nav-left {
  display: flex;
  align-items: center;
  gap: $spacing-2xl;
}

.landing-nav-logo {
  display: flex;
  align-items: center;
  padding: 0;
  background: none;
  height: 2rem;

  &:hover {
    background: none;
  }

  img {
    height: 100%;
    width: auto;
  }
}

.landing-nav-link {
  color: $color-gray-300;
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s ease;
  padding: 0;
  background: none;
  height: 2rem;
  display: flex;
  align-items: center;

  &:hover {
    color: white;
    text-decoration: underline;
  }

  &.router-link-active {
    text-decoration: underline;
  }

  @media (prefers-color-scheme: light) {
    color: $color-gray-700;

    &:hover {
      color: $color-gray-900;
    }
  }

  @media (max-width: 768px) {
    display: none !important;
  }
}

.landing-nav-right {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
}

.landing-nav-button {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  padding: $spacing-sm $spacing-md;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: $border-radius;
  color: $color-gray-200;
  font-size: $font-size-sm;
  cursor: pointer;
  transition: all 0.2s ease;
  text-decoration: none;

  &:hover {
    background: rgba(255, 255, 255, 0.15);
    border-color: rgba(255, 255, 255, 0.3);
  }

  svg {
    width: 1rem;
    height: 1rem;
  }

  @media (prefers-color-scheme: light) {
    background: rgba(0, 0, 0, 0.05);
    border-color: rgba(0, 0, 0, 0.1);
    color: $color-gray-700;

    &:hover {
      background: rgba(0, 0, 0, 0.08);
      border-color: rgba(0, 0, 0, 0.15);
    }
  }

  @media (max-width: 768px) {
    display: none !important;
  }
}

.version-container {
  position: relative;

  @media (max-width: 768px) {
    display: none !important;
  }
}

.landing-nav-version {
  padding: $spacing-sm $spacing-md;
  background: rgba(255, 255, 255, 0.05);
  border-radius: $border-radius;
  color: $color-gray-400;
  font-size: $font-size-sm;
  font-weight: 300;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.2);
  }

  @media (prefers-color-scheme: light) {
    background: rgba(0, 0, 0, 0.05);
    color: $color-gray-600;
  }
}

.version-dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: $spacing-sm;
  min-width: 200px;
  background: hsl(225, 25%, 18%);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: $border-radius-lg;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  overflow: hidden;
  z-index: 1000;

  @media (prefers-color-scheme: light) {
    background: white;
    border-color: rgba(0, 0, 0, 0.1);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  }
}

.version-dropdown-loading,
.version-dropdown-empty {
  padding: $spacing-md $spacing-lg;
  color: $color-gray-400;
  font-size: $font-size-sm;
  text-align: center;
}

.version-dropdown-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  padding: $spacing-sm $spacing-lg;
  color: $color-gray-300;
  text-decoration: none;
  font-size: $font-size-sm;
  transition: background-color 0.2s ease;

  &:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  &--current {
    background: rgba(37, 99, 235, 0.2);

    &:hover {
      background: rgba(37, 99, 235, 0.3);
    }
  }

  @media (prefers-color-scheme: light) {
    color: $color-gray-700;

    &:hover {
      background: $color-gray-100;
    }

    &--current {
      background: rgba(37, 99, 235, 0.1);

      &:hover {
        background: rgba(37, 99, 235, 0.15);
      }
    }
  }
}

.version-dropdown-tag {
  font-family: monospace;
}

.version-dropdown-badge {
  padding: 2px 6px;
  background: $color-primary-600;
  color: white;
  border-radius: $border-radius;
  font-size: 0.75rem;
  font-weight: 500;
}

.landing-nav-hamburger {
  display: none;
  cursor: pointer;
  padding: $spacing-sm;
  border-radius: $border-radius;
  background: none;
  border: none;
  color: $color-gray-300;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;

  &:hover {
    color: white;
    background-color: rgba(255, 255, 255, 0.1);
  }

  svg {
    width: 24px;
    height: 24px;
  }

  @media (prefers-color-scheme: light) {
    color: $color-gray-700;

    &:hover {
      color: $color-gray-900;
      background-color: $color-gray-100;
    }
  }

  @media (max-width: 768px) {
    display: block;
  }
}

// Mobile Menu
.mobile-menu {
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 9999;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.3s ease;

  &--open {
    opacity: 1;
    pointer-events: all;
  }

  @media (max-width: 768px) {
    display: block;
  }
}

.mobile-menu-content {
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  width: 300px;
  max-width: 85vw;
  background: hsl(225, 25%, 15%);
  box-shadow: 2px 0 12px rgba(0, 0, 0, 0.3);
  display: flex;
  flex-direction: column;
  transition: transform 0.3s ease;
  overflow: hidden;

  @media (prefers-color-scheme: light) {
    background: white;
  }
}

.mobile-menu-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-xl $spacing-lg;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  flex-shrink: 0;

  @media (prefers-color-scheme: light) {
    border-bottom-color: rgba(0, 0, 0, 0.1);
  }
}

.mobile-menu-title {
  font-size: $font-size-lg;
  font-weight: 600;
  color: white;

  @media (prefers-color-scheme: light) {
    color: $color-gray-900;
  }
}

.mobile-menu-close {
  background: none;
  border: none;
  color: $color-gray-400;
  cursor: pointer;
  padding: $spacing-sm;
  border-radius: $border-radius;
  transition:
    color 0.2s ease,
    background-color 0.2s ease;

  &:hover {
    color: white;
    background-color: rgba(255, 255, 255, 0.1);
  }

  svg {
    width: 24px;
    height: 24px;
  }

  @media (prefers-color-scheme: light) {
    color: $color-gray-600;

    &:hover {
      color: $color-gray-900;
      background-color: $color-gray-100;
    }
  }
}

.mobile-menu-nav {
  flex: 1 1 auto;
  overflow-y: auto;
  overflow-x: hidden;
  padding: $spacing-md 0;
}

.mobile-menu-link {
  display: flex;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-lg $spacing-xl;
  color: $color-gray-300;
  text-decoration: none;
  border: none;
  background: none;
  width: 100%;
  text-align: left;
  transition: all 0.2s ease;
  font-size: $font-size-lg;
  font-weight: 500;
  min-height: 56px;
  cursor: pointer;

  &:hover {
    background: $color-primary-800;
    color: white;
  }

  svg {
    width: 24px;
    height: 24px;
    flex-shrink: 0;
  }

  @media (prefers-color-scheme: light) {
    color: $color-gray-700;

    &:hover {
      background: $color-primary-100;
      color: $color-primary-900;
    }
  }
}

.mobile-menu-footer {
  margin-top: auto;
  padding: 0;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  flex-shrink: 0;

  @media (prefers-color-scheme: light) {
    border-top-color: rgba(0, 0, 0, 0.1);
  }
}

.mobile-menu-divider {
  height: 1px;
  background: rgba(255, 255, 255, 0.1);
  margin: $spacing-sm $spacing-xl;

  @media (prefers-color-scheme: light) {
    background: rgba(0, 0, 0, 0.1);
  }
}

.mobile-menu-version {
  display: block;
  padding: $spacing-lg $spacing-xl;
  font-size: $font-size-sm;
  color: $color-gray-500;
}
</style>
