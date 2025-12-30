<template>
  <div class="service-icons-container">
    <div class="service-icons">
      <template v-for="icon in icons" :key="icon.name">
        <a v-if="icon.enabled" :href="icon.href" class="service-icon-button">
          <div class="service-icon-bg">
            <component :is="icon.component" />
          </div>
          <span class="service-icon-label">{{ icon.label }}</span>
        </a>
        <div
          v-else
          class="service-icon-button service-icon-button--disabled"
          @click="handleDisabledClick"
        >
          <div class="service-icon-bg">
            <component :is="icon.component" />
          </div>
          <span class="service-icon-label">{{ icon.label }}</span>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { ServiceIcon } from '@/types/service_icon'

defineProps<{
  icons: ServiceIcon[]
}>()

const handleDisabledClick = () => {
  alert('Coming soon!')
}
</script>

<style lang="scss" scoped>
@use 'sass:color';

.service-icons-container {
  margin-bottom: $spacing-2xl;
  padding: $spacing-3xl $spacing-2xl;
  background: color.scale($theme-palette-bg-inverse, $alpha: -95%);
  backdrop-filter: blur(10px);
  border: 1px solid color.scale($theme-palette-bg-inverse, $alpha: -90%);
  border-radius: 24px;

  @media (prefers-color-scheme: light) {
    background: color.scale($theme-palette-bg-primary, $alpha: -97%);
    border-color: color.scale($theme-palette-bg-primary, $alpha: -92%);
  }
}

.service-icons {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: $spacing-2xl;
  justify-items: center;

  @media (min-width: 1024px) {
    grid-template-columns: repeat(5, 1fr);
    gap: $spacing-3xl;
  }

  @media (max-width: 768px) {
    grid-template-columns: repeat(3, 1fr);
    gap: $spacing-lg;
  }

  @media (max-width: 480px) {
    grid-template-columns: repeat(2, 1fr);
  }
}

.service-icon-button {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: $spacing-md;
  text-decoration: none;
  transition: background-color 0.2s ease;
  padding: $spacing-md;
  border-radius: $border-radius-lg;
  cursor: pointer;

  &:hover {
    background-color: color.scale($theme-palette-bg-inverse, $alpha: -95%);

    @media (prefers-color-scheme: light) {
      background-color: color.scale($theme-palette-bg-primary, $alpha: -97%);
    }

    .service-icon-bg {
      background: color.scale($theme-palette-bg-inverse, $alpha: -85%);
      border-color: color.scale($theme-palette-bg-inverse, $alpha: -70%);

      @media (prefers-color-scheme: light) {
        background: color.scale($theme-palette-bg-primary, $alpha: -92%);
        border-color: color.scale($theme-palette-bg-primary, $alpha: -85%);
      }
    }
  }

  &--disabled {
    opacity: 0.4;
  }
}

.service-icon-bg {
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color.scale($theme-palette-bg-inverse, $alpha: -90%);
  backdrop-filter: blur(10px);
  border: 1px solid color.scale($theme-palette-bg-inverse, $alpha: -80%);
  border-radius: $border-radius-lg;
  transition: all 0.2s ease;

  svg {
    width: 28px;
    height: 28px;
    color: $theme-palette-text-primary;

    @media (prefers-color-scheme: light) {
      color: $theme-palette-text-inverse;
    }
  }

  @media (prefers-color-scheme: light) {
    background: color.scale($theme-palette-bg-primary, $alpha: -95%);
    border-color: color.scale($theme-palette-bg-primary, $alpha: -90%);
  }

  @media (max-width: 768px) {
    width: 48px;
    height: 48px;

    svg {
      width: 22px;
      height: 22px;
    }
  }
}

.service-icon-label {
  font-size: $theme-font-size-sm;
  color: $theme-palette-text-muted;
  font-weight: 500;
  text-align: center;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-inverse;
  }
}
</style>
