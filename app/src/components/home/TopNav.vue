<template>
  <!-- Flash banner for minimal mode changes -->
  <FlashBanner :show="showBanner" @hide="showBanner = false">
    {{
      bannerMessage ||
      (isMinimal
        ? 'Fullscreen mode enabled' +
          `${minimizeKeyCombo ? ` (${toKeyComboString(minimizeKeyCombo)})` : ''}`
        : 'Fullscreen mode disabled' +
          `${minimizeKeyCombo ? ` (${toKeyComboString(minimizeKeyCombo)})` : ''}`)
    }}
  </FlashBanner>
  <nav class="landing-nav" :class="{ 'landing-nav--minimal': isMinimal }">
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
      <!-- TODO: This is incomplete is why it is hidden -->
      <!-- <RouterLink to="/settings" class="landing-nav-button" title="Settings">
        <SettingsIcon />
      </RouterLink> -->
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
          <div v-if="loadingReleases" class="version-dropdown-loading">
            Loading...
          </div>
          <template v-else-if="releases.length > 0">
            <button
              v-for="release in releases"
              :key="release.tagName"
              :disabled="release.isCurrentVersion || !!updatingVersion"
              @click="handleUpdate(release.tagName)"
              :class="[
                'version-dropdown-item',
                { 'version-dropdown-item--current': release.isCurrentVersion },
              ]"
            >
              <span class="version-dropdown-tag">{{ release.tagName }}</span>
              <span
                v-if="release.isCurrentVersion"
                class="version-dropdown-badge"
                >Current</span
              >
              <span
                v-if="updatingVersion === release.tagName"
                style="margin-left: 8px"
                >Updating...</span
              >
            </button>
          </template>
          <div v-else class="version-dropdown-empty">No releases available</div>
        </div>
      </div>
      <RouterLink to="/devices" class="landing-nav-button" title="Devices">
        <DeviceIcon />
        <span>Devices</span>
      </RouterLink>
      <button
        class="landing-nav-hamburger"
        @click="toggleMobileMenu"
        aria-label="Menu"
      >
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
        <button
          class="mobile-menu-close"
          @click="closeMobileMenu"
          aria-label="Close menu"
        >
          <CloseIcon />
        </button>
      </div>
      <nav class="mobile-menu-nav">
        <RouterLink to="/" class="mobile-menu-link" @click="closeMobileMenu">
          <HomeIcon />
          <span>Home</span>
        </RouterLink>
        <RouterLink
          to="/cirrus"
          class="mobile-menu-link"
          @click="closeMobileMenu"
        >
          <FolderIcon />
          <span>Cirrus</span>
        </RouterLink>
        <RouterLink
          to="/photos"
          class="mobile-menu-link"
          @click="closeMobileMenu"
        >
          <PhotoIcon />
          <span>Photos</span>
        </RouterLink>
        <!-- TODO: This is incomplete is why it is hidden -->
        <!-- <RouterLink
          to="/books"
          class="mobile-menu-link"
          @click="closeMobileMenu"
        >
          <BookIcon />
          <span>Books</span>
        </RouterLink> -->
        <RouterLink
          to="/devices"
          class="mobile-menu-link"
          @click="closeMobileMenu"
        >
          <DeviceIcon />
          <span>Devices</span>
        </RouterLink>
      </nav>
      <div class="mobile-menu-footer">
        <!-- TODO: This is incomplete is why it is hidden -->
        <!-- <RouterLink
          to="/settings"
          class="mobile-menu-link"
          title="Settings"
          @click="closeMobileMenu"
        >
          <SettingsIcon />
          <span>Settings</span>
        </RouterLink> -->
        <div class="mobile-menu-divider" />
        <span class="mobile-menu-version">Version: {{ currentVersion }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import FlashBanner from '@/components/common/FlashBanner.vue';
import VersionService, { type Release } from '@/services/versionService';
import type { NavLink } from '@/types/nav_link';
import { toKeyComboString, type KeyCombo } from '@/util/keycombo';
import { onMounted, onUnmounted, ref, watch } from 'vue';
import { RouterLink } from 'vue-router';
import CloseIcon from '../icons/CloseIcon.vue';
import DeviceIcon from '../icons/DeviceIcon.vue';
import FolderIcon from '../icons/FolderIcon.vue';
import HamburgerIcon from '../icons/HamburgerIcon.vue';
import HomeIcon from '../icons/HomeIcon.vue';
import PhotoIcon from '../icons/PhotoIcon.vue';

const props = defineProps<{
  isMinimal?: boolean;
  minimizeKeyCombo?: KeyCombo;
  navLinks?: NavLink[];
}>();
// --- Flash banner logic ---
const showBanner = ref(false);
const bannerMessage = ref('');
let bannerTimeout: ReturnType<typeof setTimeout> | null = null;

watch(
  () => props.isMinimal,
  () => {
    showBanner.value = true;
    if (bannerTimeout) clearTimeout(bannerTimeout);
    bannerTimeout = setTimeout(() => {
      showBanner.value = false;
    }, 1200);
  },
);
// --- End flash banner logic ---

const mobileMenuOpen = ref(false);
const versionDropdownOpen = ref(false);
const currentVersion = ref('vX.Y.Z');
const releases = ref<Release[]>([]);
const loadingReleases = ref(false);
const updatingVersion = ref<string | null>(null);

// Fetch current version on mount
onMounted(async () => {
  currentVersion.value = await VersionService.getCurrentVersion();

  // Add click outside listener for version dropdown
  document.addEventListener('click', handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
  if (bannerTimeout) clearTimeout(bannerTimeout);
});

const handleClickOutside = (event: MouseEvent) => {
  const container = document.getElementById('version-container');
  if (container && !container.contains(event.target as Node)) {
    versionDropdownOpen.value = false;
  }
};

const toggleVersionDropdown = async () => {
  if (versionDropdownOpen.value) {
    versionDropdownOpen.value = false;
    return;
  }

  versionDropdownOpen.value = true;
  if (releases.value.length === 0) {
    loadingReleases.value = true;
    releases.value = await VersionService.getAvailableReleases();
    loadingReleases.value = false;
  }
};

const handleUpdate = async (version: string) => {
  updatingVersion.value = version;
  try {
    bannerMessage.value = `Update to ${version} started. The server will restart once it downloads.`;
    showBanner.value = true;
    await VersionService.doUpdate(version);
  } catch (err) {
    showBanner.value = true;
    bannerMessage.value = `Update failed: ${err}`;
  } finally {
    updatingVersion.value = null;
  }
};

const toggleMobileMenu = () => {
  mobileMenuOpen.value = !mobileMenuOpen.value;
};

const closeMobileMenu = () => {
  mobileMenuOpen.value = false;
};
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
  background: hsl(from $theme-palette-bg-nav h s l / 0.97);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid hsl(from $theme-palette-bg-primary h s l / 0.1);
  width: 100vw;
  margin-left: calc(50% - 50vw);
  margin-right: calc(50% - 50vw);
  left: 0;
  right: 0;

  @media (prefers-color-scheme: light) {
    background: hsl(from $theme-palette-bg-nav h s l / 0.97);
    border-bottom-color: hsl(from $theme-palette-bg-inverse h s l / 0.1);
  }
}

.landing-nav-left {
  display: flex;
  align-items: center;
  gap: $spacing-2xl;
}

.landing-nav-link {
  color: $theme-palette-text-primary;
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s ease;
  padding: 0;
  background: none;
  height: 2rem;
  display: flex;
  align-items: center;

  &:hover {
    color: $theme-palette-accent;
    text-decoration: underline;
  }

  &.router-link-active {
    text-decoration: underline;
  }

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-primary;
    &:hover {
      color: $theme-palette-accent;
    }
  }

  @media (max-width: 768px) {
    display: none !important;
  }
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

.landing-nav--minimal {
  display: none;
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
  background: hsl(from $theme-palette-bg-primary h s l / 0.1);
  border: 1px solid $theme-palette-border;
  border-radius: $border-radius;
  color: $theme-palette-text-secondary;
  font-size: $theme-font-size-sm;
  cursor: pointer;
  transition: all 0.2s ease;
  text-decoration: none;

  &:hover {
    background: hsl(from $theme-palette-accent h s l / 0.12);
    border-color: $theme-palette-accent;
    color: $theme-palette-accent;
  }

  svg {
    width: 1rem;
    height: 1rem;
  }

  @media (prefers-color-scheme: light) {
    background: hsl(from $theme-palette-bg-inverse h s l / 0.05);
    border-color: hsl(from $theme-palette-bg-inverse h s l / 0.1);
    color: $theme-palette-text-secondary;

    &:hover {
      background: hsl(from $theme-palette-bg-inverse h s l / 0.08);
      border-color: hsl(from $theme-palette-bg-inverse h s l / 0.15);
    }
  }

  @media (max-width: 768px) {
    display: none !important;
  }
}

.version-container {
  border: 1px solid $theme-palette-border;
  border-radius: $border-radius;
  transition:
    background 0.2s,
    border-color 0.2s;
  &:hover,
  &:focus-within {
    background: hsl(from $theme-palette-accent h s l / 0.12);
    border-color: $theme-palette-accent;
  }
  position: relative;

  @media (max-width: 768px) {
    display: none !important;
  }
}

.landing-nav-version {
  padding: $spacing-sm $spacing-md;
  background: hsl(from $theme-palette-bg-primary h s l / 0.05);
  border-radius: $border-radius;
  color: $theme-palette-text-muted;
  font-size: $theme-font-size-sm;
  font-weight: 300;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background: rgba($theme-palette-bg-primary, 0.1);
    border-color: rgba($theme-palette-bg-primary, 0.2);
  }

  @media (prefers-color-scheme: light) {
    background: rgba($theme-palette-bg-inverse, 0.05);
    color: $theme-palette-text-secondary;
  }
}

.landing-nav-hamburger {
  display: none;
  cursor: pointer;
  padding: $spacing-sm;
  border-radius: $border-radius;
  background: none;
  border: none;
  color: $theme-palette-text-muted;
  transition:
    background-color 0.2s ease,
    color 0.2s ease;

  &:hover {
    color: $theme-palette-text-inverse;
    background-color: rgba($theme-palette-bg-primary, 0.1);
  }

  svg {
    width: 24px;
    height: 24px;
  }

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-secondary;

    &:hover {
      color: $theme-palette-text-primary;
      background-color: $theme-palette-bg-secondary;
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
  background: rgba($theme-palette-bg-inverse, 0.5);
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
  /* Use primary app background so the mobile menu matches the theme */
  background: $theme-palette-bg-primary;
  box-shadow: 2px 0 24px rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
  transition: transform 0.3s ease;
  overflow: hidden;

  @media (prefers-color-scheme: light) {
    background: $theme-palette-bg-primary;
  }
}

.mobile-menu-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: $spacing-xl $spacing-lg;
  border-bottom: 1px solid $theme-palette-border;
  flex-shrink: 0;

  @media (prefers-color-scheme: light) {
    border-bottom-color: rgba($theme-palette-bg-inverse, 0.1);
  }
}

.mobile-menu-title {
  font-size: $theme-font-size-lg;
  font-weight: 600;
  color: $theme-palette-text-primary;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-primary;
  }
}

.mobile-menu-close {
  background: none;
  border: none;
  color: $theme-palette-text-muted;
  cursor: pointer;
  padding: $spacing-sm;
  border-radius: $border-radius;
  transition:
    color 0.2s ease,
    background-color 0.2s ease;

  &:hover {
    color: $theme-palette-text-inverse;
    background-color: rgba($theme-palette-bg-primary, 0.1);
  }

  svg {
    width: 24px;
    height: 24px;
  }

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-secondary;

    &:hover {
      color: $theme-palette-text-primary;
      background-color: $theme-palette-bg-secondary;
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
  color: $theme-palette-text-primary;
  text-decoration: none;
  border: none;
  background: none;
  width: 100%;
  text-align: left;
  transition: all 0.2s ease;
  font-size: $theme-font-size-lg;
  font-weight: 500;
  min-height: 56px;
  cursor: pointer;

  &:hover {
    background: $theme-palette-accent;
    color: $theme-palette-text-inverse;
  }

  svg {
    width: 24px;
    height: 24px;
    flex-shrink: 0;
  }

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-secondary;

    &:hover {
      background: $theme-palette-accent-hover;
      color: $theme-palette-accent;
    }
  }
}

.mobile-menu-footer {
  margin-top: auto;
  padding: 0;
  border-top: 1px solid $theme-palette-border;
  flex-shrink: 0;

  @media (prefers-color-scheme: light) {
    border-top-color: rgba($theme-palette-bg-inverse, 0.1);
  }
}

.mobile-menu-divider {
  height: 1px;
  background: rgba($theme-palette-bg-primary, 0.1);
  margin: $spacing-sm $spacing-xl;

  @media (prefers-color-scheme: light) {
    background: rgba($theme-palette-bg-inverse, 0.1);
  }
}

.mobile-menu-version {
  display: block;
  padding: $spacing-lg $spacing-xl;
  font-size: $theme-font-size-sm;
  color: $theme-palette-text-muted;
}

.version-dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: $spacing-sm;
  min-width: 200px;
  background: $theme-palette-bg-nav;
  border: 1px solid hsl(from $theme-palette-border h s l / 0.1);
  border-radius: $border-radius-lg;
  box-shadow: 0 4px 12px hsl(from $theme-palette-border h s l / 0.3);
  overflow: hidden;
  z-index: 1000;
}

.version-dropdown-badge {
  padding: 2px 6px;
  background: $theme-palette-accent;
  color: $theme-palette-text-inverse;
  border-radius: $border-radius;
  font-size: 0.75rem;
  font-weight: 500;
}

.version-dropdown-loading,
.version-dropdown-empty {
  padding: $spacing-md $spacing-lg;
  color: $theme-palette-text-muted;
  font-size: $theme-font-size-sm;
  text-align: center;
}

.version-dropdown-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: $spacing-md;
  padding: $spacing-sm $spacing-lg;
  background: $theme-palette-bg-nav;
  border: none;
  border-radius: $border-radius;
  color: $theme-palette-text-muted;
  font-size: $theme-font-size-sm;
  text-decoration: none;
  cursor: pointer;
  transition:
    background-color 0.2s,
    color 0.2s;

  &:hover {
    background: $theme-palette-accent;
    color: $theme-palette-text-inverse;
  }

  &--current {
    background: hsl(from $theme-palette-accent h s l / 0.2);
    cursor: default;
    color: $theme-palette-text-primary;
    &:hover {
      background: hsl(from $theme-palette-accent h s l / 0.3);
    }
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-inverse;
    background: $theme-palette-bg-nav;

    &:hover {
      background: $theme-palette-accent-hover;
      color: $theme-palette-text-inverse;
    }

    &--current {
      background: hsl(from $theme-palette-accent h s l / 0.1);
      color: $theme-palette-text-inverse;
      &:hover {
        background: hsl(from $theme-palette-accent h s l / 0.15);
      }
    }
  }
}

.version-dropdown-tag {
  font-family: monospace;
}
</style>
