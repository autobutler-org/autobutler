<template>
  <div class="device-card">
    <div class="device-card-header">
      <DeviceCardIcon />
      <div class="device-card-title-section">
        <h3 class="device-card-title">
          {{ displayedDevice.name }}
        </h3>
        <p class="device-card-type">
          {{ displayedDevice.is_internal ? 'Internal' : 'External' }}
          <span v-if="!displayedDevice.is_internal"
            >• Mounted?
            <ToggleSwitch
              :model-value="!!displayedDevice.usb_info?.mountPath"
              @update:model-value="onToggleUsbMount"
            />
          </span>
        </p>
      </div>
    </div>
    <div
      class="device-card-body"
      @click="goToCirrus"
      v-if="displayedDevice.is_internal || displayedDevice.usb_info?.mountPath"
    >
      <StoragePartition :device="device" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import DeviceCardIcon from '@/components/icons/DeviceCardIcon.vue';
import DevicesService from '@/services/devicesService';
import type { Device } from '@/types/device';
import { ref } from 'vue';
import StoragePartition from './StoragePartition.vue';
import ToggleSwitch from './ToggleSwitch.vue';
const onToggleUsbMount = async (checked: boolean) => {
  const serial = displayedDevice.value.usb_info?.serial;
  if (!serial) return;
  if (checked) {
    await enableDevice(serial);
  } else {
    await disableDevice(serial);
  }
};

const props = defineProps<{
  device: Device;
}>();

const displayedDevice = ref<Device>(props.device);

const disableDevice = async (serial: string) => {
  try {
    await DevicesService.disableUsbStorageDevice(serial);
    // Refresh device
    const data = await DevicesService.getDeviceStatus(serial);
    displayedDevice.value = data;
  } catch (e) {
    console.error('Failed to disable device:', e);
  }
};

const enableDevice = async (serial: string) => {
  try {
    await DevicesService.enableUsbStorageDevice(serial);
    // Refresh device
    const data = await DevicesService.getDeviceStatus(serial);
    displayedDevice.value = data;
  } catch (e) {
    console.error('Failed to disable device:', e);
  }
};

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
  min-width: 16.25rem;
  max-width: 37.5rem;
  flex: 1 1 20rem;
  margin: 0;
}

.device-card-body {
  margin-top: 0.5rem;
  cursor: pointer;
}

.device-card-usb {
  margin-top: 0.8rem;
}

.device-card-header {
  display: flex;
  align-items: center;
  gap: 1.5rem;
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

.device-disable-btn {
  background: $theme-palette-bg-secondary;
  color: $theme-palette-text-muted;
  border: none;
  border-radius: $border-radius;
  padding: 4px 16px;
  font-size: $theme-font-size-sm;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}
.device-disable-btn:hover {
  background: $theme-palette-border;
  color: $theme-palette-text-primary;
}
.device-enable-btn {
  background: $theme-palette-accent;
  color: $theme-palette-text-inverse;
  border: none;
  border-radius: $border-radius;
  padding: 4px 16px;
  font-size: $theme-font-size-sm;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}
.device-enable-btn:hover {
  background: $theme-palette-accent-hover;
}
</style>
