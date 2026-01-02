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
import type { ServiceIcon } from '@/types/service_icon';

defineProps<{
  icons: ServiceIcon[];
}>();

const handleDisabledClick = () => {
  alert('Coming soon!');
};
</script>

<style lang="scss" scoped>
.service-icons-container {
  margin-bottom: $spacing-2xl;
  padding: $spacing-md $spacing-md;
  background: hsl(from $theme-palette-bg-inverse h s l / 0.05);
  backdrop-filter: blur(10px);
  border: 1px solid hsl(from $theme-palette-bg-inverse h s l / 0.1);
  border-radius: 24px;

  @media (prefers-color-scheme: light) {
    background: hsl(from $theme-palette-bg-primary h s l / 0.03);
    border-color: hsl(from $theme-palette-bg-primary h s l / 0.08);
  }
}

.service-icons {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  justify-items: center;

  @media (min-width: 1024px) {
    grid-template-columns: repeat(5, 1fr);
  }

  @media (max-width: 768px) {
    grid-template-columns: repeat(3, 1fr);
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
    background-color: hsl(from $theme-palette-bg-inverse h s l / 0.05);

    @media (prefers-color-scheme: light) {
      background-color: hsl(from $theme-palette-bg-primary h s l / 0.03);
    }

    .service-icon-bg {
      background: hsl(from $theme-palette-bg-inverse h s l / 0.15);
      border-color: hsl(from $theme-palette-bg-inverse h s l / 0.3);

      @media (prefers-color-scheme: light) {
        background: hsl(from $theme-palette-bg-primary h s l / 0.08);
        border-color: hsl(from $theme-palette-bg-primary h s l / 0.15);
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
  background: hsl(from $theme-palette-bg-inverse h s l / 0.1);
  backdrop-filter: blur(10px);
  border: 1px solid hsl(from $theme-palette-bg-inverse h s l / 0.2);
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
    background: hsl(from $theme-palette-bg-primary h s l / 0.05);
    border-color: hsl(from $theme-palette-bg-primary h s l / 0.1);
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
