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
</style>
