<template>
  <div
    class="device-card"
    :class="{
      'device-card--disabled': !displayedDevice.isEnabled,
    }"
  >
    <div class="device-card-header">
      <DeviceCardIcon />
      <div class="device-card-title-section">
        <h3 class="device-card-title">
          {{ displayedDevice.name }}
        </h3>
        <p class="device-card-type">
          {{ displayedDevice.isInternal ? 'Internal' : 'External' }}
          <span v-if="!displayedDevice.isInternal && displayedDevice.isEnabled"
            >• Mounted?
            <ToggleSwitch
              :model-value="!!displayedDevice.usbInfo?.mountPath"
              @update:model-value="onToggleUsbMount"
            />
          </span>
        </p>
      </div>
    </div>
    <div
      v-if="!displayedDevice.isEnabled"
      class="device-card-body device-card-body--disabled"
      @click="showEnableModal = true"
    >
      <div class="device-disabled-message">
        <p>Device not enabled</p>
        <p class="device-disabled-hint">Click to enable this device</p>
      </div>
      <div class="storage-bar-disabled" />
    </div>
    <div v-else-if="minimal" class="device-card-body device-card-body--minimal">
      <div class="device-card-minimal">
        <div class="device-card-title">{{ displayedDevice.name }}</div>
        <div class="device-card-type">{{ displayedDevice.isInternal ? 'Internal' : 'External' }}</div>
      </div>
    </div>
    <div v-else class="device-card-body" @click="goToCirrus">
      <StoragePartition :device="displayedDevice" />
    </div>
  </div>

  <!-- Enable Device Modal -->
  <Teleport to="body">
    <div
      v-if="showEnableModal"
      class="modal-overlay"
      @click.self="showEnableModal = false"
    >
      <div class="modal-content">
        <h2>Enable Device</h2>
        <p>Do you want to enable "{{ displayedDevice.name }}"?</p>
        <p class="modal-hint">
          This will mount the device and allow Autobutler to access its
          contents.
        </p>
        <div class="modal-actions">
          <button
            class="modal-btn modal-btn--secondary"
            @click="showEnableModal = false"
            :disabled="isEnabling"
          >
            Cancel
          </button>
          <button
            class="modal-btn modal-btn--primary"
            @click="handleEnableDevice"
            :disabled="isEnabling"
          >
            {{ isEnabling ? 'Enabling...' : 'Enable Device' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import DeviceCardIcon from '@/components/icons/DeviceCardIcon.vue';
import DevicesService from '@/services/devicesService';
import type { Device } from '@/types/device';
import { ref } from 'vue';
import StoragePartition from './StoragePartition.vue';
import ToggleSwitch from './ToggleSwitch.vue';

const props = defineProps<{
  device: Device;
  minimal?: boolean;
}>();

const displayedDevice = ref<Device>(props.device);
const minimal = props.minimal || false;

const showEnableModal = ref(false);
const isEnabling = ref(false);

const onToggleUsbMount = async (checked: boolean) => {
  const serial = displayedDevice.value.usbInfo?.serial;
  if (!serial) return;
  if (checked) {
    await enableDevice(serial);
  } else {
    await disableDevice(serial);
  }
};

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
    console.error('Failed to enable device:', e);
  }
};

const handleEnableDevice = async () => {
  const serial = displayedDevice.value.usbInfo?.serial;
  if (!serial) return;

  isEnabling.value = true;
  try {
    await enableDevice(serial);
    showEnableModal.value = false;
  } catch (e) {
    console.error('Failed to enable device:', e);
  } finally {
    isEnabling.value = false;
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

.device-card--disabled {
  background: rgba($theme-palette-bg-secondary, 0.5);
  opacity: 0.7;

  .device-card-title {
    color: $theme-palette-text-muted;
  }
}

.device-card--backup {
  border: 2px solid rgba(255, 255, 255, 0.2);
  box-shadow:
    0 2px 12px 0 rgba($theme-palette-bg-nav, 0.15),
    0 0 0 1px rgba(255, 255, 255, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);

  @media (prefers-color-scheme: light) {
    border: 2px solid rgba(255, 255, 255, 0.5);
    box-shadow:
      0 2px 12px 0 rgba(0, 0, 0, 0.08),
      0 0 0 1px rgba(255, 255, 255, 0.3),
      inset 0 1px 0 rgba(255, 255, 255, 0.5);
  }
}

.device-card-body {
  margin-top: 0.5rem;
  cursor: pointer;
}

.device-card-body--disabled {
  cursor: pointer;

  &:hover {
    opacity: 0.9;
  }
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

// Disabled device styles
.device-disabled-message {
  text-align: center;
  padding: 2rem 0;

  p {
    margin: 0.5rem 0;
    color: $theme-palette-text-muted;
  }

  .device-disabled-hint {
    font-size: $theme-font-size-sm;
    color: $theme-palette-text-muted;
    opacity: 0.7;
  }
}

.storage-bar-disabled {
  height: 8px;
  background: repeating-linear-gradient(
    45deg,
    $theme-palette-border,
    $theme-palette-border 10px,
    $theme-palette-bg-secondary 10px,
    $theme-palette-bg-secondary 20px
  );
  border-radius: 4px;
  opacity: 0.5;
  margin-top: 1rem;
}

// Modal styles
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: $theme-palette-bg-primary;
  border-radius: 12px;
  padding: 2rem;
  max-width: 500px;
  width: 90%;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.3);

  h2 {
    margin: 0 0 1rem 0;
    color: $theme-palette-text-primary;
    font-size: $theme-font-size-xl;
  }

  p {
    color: $theme-palette-text-primary;
    margin: 0.5rem 0;
  }

  .modal-hint {
    font-size: $theme-font-size-sm;
    color: $theme-palette-text-muted;
    margin-bottom: 1.5rem;
  }
}

.modal-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
}

.modal-btn {
  padding: 0.5rem 1.5rem;
  border-radius: $border-radius;
  border: none;
  font-size: $theme-font-size-base;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.modal-btn--secondary {
  background: $theme-palette-bg-secondary;
  color: $theme-palette-text-primary;

  &:hover:not(:disabled) {
    background: $theme-palette-border;
  }
}

.modal-btn--primary {
  background: $theme-palette-accent;
  color: $theme-palette-text-inverse;

  &:hover:not(:disabled) {
    background: $theme-palette-accent-hover;
  }
}
</style>
