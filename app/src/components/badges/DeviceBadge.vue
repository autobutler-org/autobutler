<template>
  <div class="device-badge-container" :title="'Device: ' + deviceName">
    <DeviceIcon />
    <span class="device-badge-name">{{ deviceName }}</span>
  </div>
</template>

<script lang="ts" setup>
import DeviceIcon from '@/components/icons/DeviceIcon.vue';

defineProps<{
  deviceName: string;
}>();
</script>

<style lang="scss" scoped>
.device-badge-container {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  margin-top: $spacing-xs;
  padding: 2px 6px;
  background-color: $theme-palette-accent;
  border: 1px solid $theme-palette-border-strong;
  border-radius: $border-radius-sm;
  font-size: $theme-font-size-xs;
  color: $theme-palette-text-inverse;
  white-space: nowrap;

  &:hover {
    background-color: $theme-palette-accent-hover;
  }
}

.device-badge-name {
  font-weight: 500;
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
}

// Improve readability on small/mobile screens
@media (max-width: 640px) {
  .device-badge-container {
    padding: 4px 8px;
    gap: $spacing-xs;
    font-size: $theme-font-size-sm;
  }

  .device-badge-name {
    max-width: 120px;
  }
}

// On very small screens, collapse badge to icon-only to avoid overlaying text
@media (max-width: 420px) {
  .device-badge-container {
    padding: 4px;
    gap: 0.25rem;
    border-radius: $border-radius;
  }

  .device-badge-name {
    max-width: 5ch;
  }
}
</style>
