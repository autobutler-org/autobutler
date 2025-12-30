<template>
  <div class="hero">
    <div class="hero-content">
      <!-- Hero Icon -->
      <div class="hero-icon">
        <img src="/img/butler.png" alt="AutoButler Logo" />
        <h1 class="hero-title">AutoButler</h1>
        <p class="hero-description">
          Your secure home cloud for files, planning, and everyday tasks —
          elegantly organized in one place.
        </p>
      </div>

      <!-- CTA Buttons -->
      <div class="hero-actions">
        <a class="btn btn--primary" href="/cirrus">
          <span>Browse files</span>
          <RightArrowIcon />
        </a>
      </div>

      <!-- Service Icons -->
      <ServiceIcons :icons="defaultServiceIcons" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { ServiceIcon } from '@/types/service_icon'
import BackupIcon from '../icons/BackupIcon.vue'
import BookIcon from '../icons/BookIcon.vue'
import DocsIcon from '../icons/DocsIcon.vue'
import EmailIcon from '../icons/EmailIcon.vue'
import FolderIcon from '../icons/FolderIcon.vue'
import PasswordsIcon from '../icons/PasswordsIcon.vue'
import PhotoIcon from '../icons/PhotoIcon.vue'
import RightArrowIcon from '../icons/RightArrowIcon.vue'
import VpnIcon from '../icons/VpnIcon.vue'
import ServiceIcons from './ServiceIcons.vue'

// Default service icons matching the Go implementation
const defaultServiceIcons: ServiceIcon[] = [
  {
    name: 'cirrus-drive',
    label: 'Cirrus Drive',
    href: '/cirrus',
    enabled: true,
    component: FolderIcon,
  },
  {
    name: 'photos',
    label: 'Photos',
    href: '/photos',
    enabled: true,
    component: PhotoIcon,
  },
  {
    name: 'books',
    label: 'Books',
    href: '/books',
    enabled: true,
    component: BookIcon,
  },
  {
    name: 'docs',
    label: 'Docs',
    href: '/cirrus',
    enabled: false,
    component: DocsIcon,
  },
  {
    name: 'passwords',
    label: 'Passwords',
    href: '#',
    enabled: false,
    component: PasswordsIcon,
  },
  {
    name: 'vpn',
    label: 'VPN',
    href: '#',
    enabled: false,
    component: VpnIcon,
  },
  {
    name: 'email',
    label: 'Email',
    href: '#',
    enabled: false,
    component: EmailIcon,
  },
  {
    name: 'backups',
    label: 'Backups',
    href: '#',
    enabled: false,
    component: BackupIcon,
  },
]
</script>

<style lang="scss" scoped>
@use 'sass:color';

.hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: $spacing-2xl;
  flex: 1;

  @media (max-width: 768px) {
    padding: $spacing-xl $spacing-md;
    padding-bottom: 100px;
    min-height: calc(100vh - 200px);
  }

  @media (max-width: 480px) {
    padding: $spacing-md $spacing-sm;
    padding-bottom: 100px;
  }
}

.hero-content {
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: $spacing-2xl;

  @media (max-width: 768px) {
    max-width: 100%;
    gap: $spacing-lg;
  }
}

.hero-icon {
  position: relative;

  img {
    width: 8rem;
    height: 8rem;
    margin: 0 auto $spacing-xl;
    display: block;

    @media (max-width: 768px) {
      width: 5rem;
      height: 5rem;
      margin-bottom: $spacing-md;
    }

    @media (max-width: 480px) {
      width: 4rem;
      height: 4rem;
    }
  }
}

.hero-title {
  font-size: $theme-font-size-5xl;
  font-weight: 800;
  margin-bottom: $spacing-lg;
  color: $theme-palette-text-primary;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-inverse;
  }

  @media (max-width: 768px) {
    font-size: $theme-font-size-3xl;
    margin-bottom: $spacing-md;
    line-height: 1.2;
  }

  @media (max-width: 480px) {
    font-size: $theme-font-size-2xl;
  }
}

.hero-description {
  font-size: $theme-font-size-xl;
  color: $theme-palette-text-muted;
  margin-bottom: $spacing-2xl;
  line-height: 1.6;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-secondary;
  }

  @media (max-width: 768px) {
    font-size: $theme-font-size-lg;
    margin-bottom: $spacing-lg;
    line-height: 1.5;
  }

  @media (max-width: 480px) {
    font-size: $theme-font-size-base;
  }
}

.hero-actions {
  display: flex;
  flex-direction: column;
  gap: $spacing-lg;
  justify-content: center;

  @media (min-width: 640px) {
    flex-direction: row;
  }

  @media (max-width: 768px) {
    width: 100%;
    gap: $spacing-md;
  }
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-sm;
  padding: $spacing-md $spacing-xl;
  border-radius: $border-radius-lg;
  font-size: $theme-font-size-base;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  border: none;
  text-decoration: none;

  &--primary {
    background: $theme-palette-accent;
    color: $theme-palette-text-inverse;
    box-shadow: 0 4px 14px color.scale($theme-palette-accent, $alpha: -70%);

    &:hover {
      background: $theme-palette-accent-hover;
      box-shadow: 0 6px 20px color.scale($theme-palette-accent, $alpha: -60%);
    }

    svg {
      width: 1.25rem;
      height: 1.25rem;
    }
  }

  @media (max-width: 768px) {
    width: 100%;
    padding: $spacing-md;
    font-size: $theme-font-size-base;
    justify-content: center;
  }
}
</style>
