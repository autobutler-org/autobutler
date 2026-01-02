<template>
  <div class="device-card" @click="goToCirrus">
    <div class="device-card-header">
      <DeviceCardIcon />
      <div class="device-card-title-section">
        <h3 class="device-card-title">
          {{ device.name }}
        </h3>
        <p class="device-card-type">
          {{ device.is_internal ? 'External' : 'Internal' }} •
          {{ device.file_system }} •
          {{ device.device_path }}
        </p>
      </div>
    </div>
    <div class="device-card-body">
      <StoragePartition :device="device" />
    </div>
    <div class="device-card-footer">
      <div class="device-card-mount">
        <span class="device-mount-label">Mount: {{ device.mount_point }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import DeviceCardIcon from '@/components/icons/DeviceCardIcon.vue';
import StoragePartition from './StoragePartition.vue';

defineProps({
  device: {
    type: Object,
    required: true,
  },
});

const goToCirrus = () => {
  window.location.href = '/cirrus';
};
</script>

<style lang="scss" scoped>
.device-card {
  background: rgba($theme-palette-bg-primary, 0.7);
  border-radius: 18px;
  box-shadow: 0 2px 12px 0 rgba($theme-palette-bg-nav, 0.15);
  padding: 2.2rem 2.2rem 1.5rem 2.2rem;
  display: flex;
  flex-direction: column;
  gap: 1.2rem;
  max-width: 600px;
  margin: 0 auto;
}

.device-card-body {
  margin-top: 0.5rem;
}

.device-card-footer {
  margin-top: 0.8rem;
}

.device-card-header {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.device-card-mount {
  font-size: $theme-font-size-base;
  color: $theme-palette-text-secondary;
}

.device-card-title {
  font-size: $theme-font-size-2xl;
  font-weight: 700;
  color: $theme-palette-text-primary;
  margin: 0;
}

.device-card-title-section {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.device-card-type {
  font-size: $theme-font-size-base;
  color: $theme-palette-text-muted;
  margin: 0;
}
</style>
