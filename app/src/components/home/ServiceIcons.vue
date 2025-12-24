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

<script setup lang="ts">
import type { ServiceIcon } from '@/types/service_icon'

defineProps<{
  icons: ServiceIcon[]
}>()

const handleDisabledClick = () => {
  alert('Coming soon!')
}
</script>

<style lang="scss" scoped>
.service-icons-container {
  margin-bottom: $spacing-2xl;
  padding: $spacing-3xl $spacing-2xl;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 24px;

  @media (prefers-color-scheme: light) {
    background: rgba(0, 0, 0, 0.03);
    border-color: rgba(0, 0, 0, 0.08);
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
    background-color: rgba(255, 255, 255, 0.05);

    @media (prefers-color-scheme: light) {
      background-color: rgba(0, 0, 0, 0.03);
    }

    .service-icon-bg {
      background: rgba(255, 255, 255, 0.15);
      border-color: rgba(255, 255, 255, 0.3);

      @media (prefers-color-scheme: light) {
        background: rgba(0, 0, 0, 0.08);
        border-color: rgba(0, 0, 0, 0.15);
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
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: $border-radius-lg;
  transition: all 0.2s ease;

  svg {
    width: 28px;
    height: 28px;
    color: white;

    @media (prefers-color-scheme: light) {
      color: $color-gray-700;
    }
  }

  @media (prefers-color-scheme: light) {
    background: rgba(0, 0, 0, 0.05);
    border-color: rgba(0, 0, 0, 0.1);
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
  font-size: $font-size-sm;
  color: $color-gray-300;
  font-weight: 500;
  text-align: center;

  @media (prefers-color-scheme: light) {
    color: $color-gray-700;
  }
}
</style>
