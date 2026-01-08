<template>
  <div class="device-card">
    <div class="device-card-header" @click="goToCirrus">
      <DeviceCardIcon />
      <div class="device-card-title-section">
        <h3 class="device-card-title">
          {{ displayedDevice.name }}
        </h3>
        <p class="device-card-type">
          {{ displayedDevice.is_internal ? 'Internal' : 'External' }}
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
    <div class="device-card-footer">
      <div v-if="displayedDevice.usb_info" class="device-card-usb-info">
        <!-- Toggle button to mount and unmount the device -->
        <!-- if mount path is "", have "enable" button -->
        <!-- if mount path is non-empty, have "disable" button -->
        <div
          class="device-card-mount"
          v-if="displayedDevice.usb_info.mountPath"
        >
          <button
            @click="disableDevice(displayedDevice.usb_info.serial)"
            class="device-disable-btn"
          >
            Disable
          </button>
        </div>
        <div class="device-card-mount" v-else>
          <button
            @click="enableDevice(displayedDevice.usb_info.serial)"
            class="device-enable-btn"
          >
            Enable
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import DeviceCardIcon from '@/components/icons/DeviceCardIcon.vue';
import DevicesService from '@/services/devicesService';
import type { Device } from '@/types/device';
import { ref } from 'vue';
import StoragePartition from './StoragePartition.vue';

const props = defineProps<{
  device: Device;
}>();

const displayedDevice = ref<Device>(props.device);

const disableDevice = async (serial: string) => {
  try {
    console.log('Disabling device with serial:', serial);
    await DevicesService.disableUsbStorageDevice(serial);
    console.log('disabled device with serial:', serial);
    // Refresh device
    const data = await DevicesService.findUsbStorageDevice(serial);
    console.log(data);
    displayedDevice.value.usb_info = data;
    displayedDevice.value.mount_point = data.mountPath;
  } catch (e) {
    console.error('Failed to disable device:', e);
  }
};

const enableDevice = async (serial: string) => {
  try {
    console.log('Enabling device with serial:', serial);
    await DevicesService.enableUsbStorageDevice(serial);
    console.log('enabled device with serial:', serial);
    // Refresh device
    const data = await DevicesService.findUsbStorageDevice(serial);
    console.log(data);
    displayedDevice.value.usb_info = data;
    displayedDevice.value.mount_point = data.mountPath;
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
  max-width: 600px;
  margin: 0 auto;
}

.device-card-body {
  margin-top: 0.5rem;
  cursor: pointer;
}

.device-card-footer {
  margin-top: 0.8rem;
}

.device-card-header {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  cursor: pointer;
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
