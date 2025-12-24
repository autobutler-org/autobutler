<template>
  <div class="device-badge-container" :title="'Device: ' + deviceName">
    <DeviceIcon />
    <span class="device-badge-name">{{ deviceName }}</span>
  </div>
</template>

<script setup lang="ts">
import DeviceIcon from '@/components/icons/DeviceIcon.vue'

defineProps<{
  deviceName: string
}>()
</script>

<style lang="scss" scoped>
.device-badge-container {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-xs;
  margin-top: $spacing-xs;
  padding: 2px 6px;
  background-color: $color-blue-50;
  border: 1px solid $color-blue-200;
  border-radius: $border-radius-sm;
  font-size: 10px;
  color: $color-blue-700;
  white-space: nowrap;

  @media (prefers-color-scheme: dark) {
    background-color: $color-blue-900;
    border-color: $color-blue-700;
    color: $color-blue-200;
  }
}

.device-badge-name {
  font-weight: 500;
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
