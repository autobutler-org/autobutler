<template>
  <DeviceIcon />
  <span class="device-badge-name">{{ deviceName }}</span>
</template>

<script setup lang="ts">
import DeviceIcon from '@/components/icons/DeviceIcon.vue'

defineProps<{
  deviceName: string
}>()
</script>

<style lang="scss" scoped>
.device-badge-name {
  font-weight: 500;
  max-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
