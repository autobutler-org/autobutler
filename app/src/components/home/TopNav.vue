<template>
  <nav class="landing-nav">
    <div class="landing-nav-left">
      <RouterLink to="/" class="landing-nav-logo">
        <img src="/img/butler.png" height="24" width="24" alt="AutoButler" />
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
        <!-- Settings icon -->
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="12" cy="12" r="3" />
          <path
            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
          />
        </svg>
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
        <!-- Devices icon -->
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
          <line x1="8" y1="21" x2="16" y2="21" />
          <line x1="12" y1="17" x2="12" y2="21" />
        </svg>
        <span>Devices</span>
      </RouterLink>
      <button class="landing-nav-hamburger" @click="toggleMobileMenu" aria-label="Menu">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <line x1="3" y1="12" x2="21" y2="12" />
          <line x1="3" y1="6" x2="21" y2="6" />
          <line x1="3" y1="18" x2="21" y2="18" />
        </svg>
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
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
      <nav class="mobile-menu-nav">
        <RouterLink to="/" class="mobile-menu-link" @click="closeMobileMenu">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
            />
          </svg>
          <span>Home</span>
        </RouterLink>
        <RouterLink to="/cirrus" class="mobile-menu-link" @click="closeMobileMenu">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
          <span>Cirrus</span>
        </RouterLink>
        <RouterLink to="/photos" class="mobile-menu-link" @click="closeMobileMenu">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
          <span>Photos</span>
        </RouterLink>
        <RouterLink to="/books" class="mobile-menu-link" @click="closeMobileMenu">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M4 19V6.2C4 5.0799 4 4.51984 4.21799 4.09202C4.40973 3.71569 4.71569 3.40973 5.09202 3.21799C5.51984 3 6.0799 3 7.2 3H16.8C17.9201 3 18.4802 3 18.908 3.21799C19.2843 3.40973 19.5903 3.71569 19.782 4.09202C20 4.51984 20 5.0799 20 6.2V17H6C4.89543 17 4 17.8954 4 19ZM4 19C4 20.1046 4.89543 21 6 21H20M9 7H15M9 11H15M19 17V21"
            />
          </svg>
          <span>Books</span>
        </RouterLink>
        <RouterLink to="/devices" class="mobile-menu-link" @click="closeMobileMenu">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <rect x="2" y="3" width="20" height="14" rx="2" ry="2" />
            <line x1="8" y1="21" x2="16" y2="21" />
            <line x1="12" y1="17" x2="12" y2="21" />
          </svg>
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
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="3" />
            <path
              d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
            />
          </svg>
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
import type { NavLink } from '@/types/home'
import { RouterLink } from 'vue-router'
import { getCurrentVersion, getAvailableReleases, type Release } from '@/services/versionService'

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

function handleClickOutside(event: MouseEvent) {
  const container = document.getElementById('version-container')
  if (container && !container.contains(event.target as Node)) {
    versionDropdownOpen.value = false
  }
}

async function toggleVersionDropdown() {
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

function toggleMobileMenu() {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

function closeMobileMenu() {
  mobileMenuOpen.value = false
}
</script>

<style lang="scss" scoped>
.landing-nav {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-xl) var(--spacing-2xl);
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
  gap: var(--spacing-2xl);
}

.landing-nav-logo {
  display: flex;
  align-items: center;
  padding: 0;
  background: none;

  &:hover {
    background: none;
  }
}

.landing-nav-link {
  color: var(--color-gray-300);
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s ease;
  padding: 0;
  background: none;

  &:hover {
    color: white;
    background: none;
  }

  @media (prefers-color-scheme: light) {
    color: var(--color-gray-700);

    &:hover {
      color: var(--color-gray-900);
    }
  }

  @media (max-width: 768px) {
    display: none !important;
  }
}

.landing-nav-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
}

.landing-nav-button {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: var(--border-radius);
  color: var(--color-gray-200);
  font-size: var(--font-size-sm);
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
    color: var(--color-gray-700);

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
  padding: var(--spacing-sm) var(--spacing-md);
  background: rgba(255, 255, 255, 0.05);
  border-radius: var(--border-radius);
  color: var(--color-gray-400);
  font-size: var(--font-size-sm);
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
    color: var(--color-gray-600);
  }
}

.version-dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: var(--spacing-sm);
  min-width: 200px;
  background: hsl(225, 25%, 18%);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: var(--border-radius-lg);
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
  padding: var(--spacing-md) var(--spacing-lg);
  color: var(--color-gray-400);
  font-size: var(--font-size-sm);
  text-align: center;
}

.version-dropdown-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-lg);
  color: var(--color-gray-300);
  text-decoration: none;
  font-size: var(--font-size-sm);
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
    color: var(--color-gray-700);

    &:hover {
      background: var(--color-gray-100);
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
  background: var(--color-primary-600);
  color: white;
  border-radius: var(--border-radius);
  font-size: 0.75rem;
  font-weight: 500;
}

.landing-nav-hamburger {
  display: none;
  cursor: pointer;
  padding: var(--spacing-sm);
  border-radius: var(--border-radius);
  background: none;
  border: none;
  color: var(--color-gray-300);
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
    color: var(--color-gray-700);

    &:hover {
      color: var(--color-gray-900);
      background-color: var(--color-gray-100);
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
  padding: var(--spacing-xl) var(--spacing-lg);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  flex-shrink: 0;

  @media (prefers-color-scheme: light) {
    border-bottom-color: rgba(0, 0, 0, 0.1);
  }
}

.mobile-menu-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: white;

  @media (prefers-color-scheme: light) {
    color: var(--color-gray-900);
  }
}

.mobile-menu-close {
  background: none;
  border: none;
  color: var(--color-gray-400);
  cursor: pointer;
  padding: var(--spacing-sm);
  border-radius: var(--border-radius);
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
    color: var(--color-gray-600);

    &:hover {
      color: var(--color-gray-900);
      background-color: var(--color-gray-100);
    }
  }
}

.mobile-menu-nav {
  flex: 1 1 auto;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--spacing-md) 0;
}

.mobile-menu-link {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg) var(--spacing-xl);
  color: var(--color-gray-300);
  text-decoration: none;
  border: none;
  background: none;
  width: 100%;
  text-align: left;
  transition: all 0.2s ease;
  font-size: var(--font-size-lg);
  font-weight: 500;
  min-height: 56px;
  cursor: pointer;

  &:hover {
    background: var(--color-primary-800);
    color: white;
  }

  svg {
    width: 24px;
    height: 24px;
    flex-shrink: 0;
  }

  @media (prefers-color-scheme: light) {
    color: var(--color-gray-700);

    &:hover {
      background: var(--color-primary-100);
      color: var(--color-primary-900);
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
  margin: var(--spacing-sm) var(--spacing-xl);

  @media (prefers-color-scheme: light) {
    background: rgba(0, 0, 0, 0.1);
  }
}

.mobile-menu-version {
  display: block;
  padding: var(--spacing-lg) var(--spacing-xl);
  font-size: var(--font-size-sm);
  color: var(--color-gray-500);
}
</style>
