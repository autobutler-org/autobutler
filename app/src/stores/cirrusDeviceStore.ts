import type { Device } from '@/types/device';
import { defineStore } from 'pinia';
import { DEFAULT_DEVICE_SERIAL } from '@/constants/device';

export const useCirrusDeviceStore = defineStore('cirrusDevice', {
  state: () => ({
    selectedDeviceSerial: DEFAULT_DEVICE_SERIAL as string,
    selectedDeviceSerials: [] as string[],
    devices: [] as Device[],
  }),
  actions: {
    setDevices(devices: Device[]) {
      // Only include devices that are internal or have a mount_point (i.e., mounted)
      const filtered = devices.filter(
        (d) => !d.usbInfo?.serial || !!d.mountPoint,
      );
      this.devices = filtered;
      // Ensure the selectedDeviceSerial is valid; if not, fallback to internal or first device
      if (
        !filtered.some((d) => d.usbInfo?.serial === this.selectedDeviceSerial)
      ) {
        const internal = filtered.find((d) => !d.usbInfo?.serial);
        this.selectedDeviceSerial = internal
          ? DEFAULT_DEVICE_SERIAL
          : filtered[0]?.usbInfo?.serial || DEFAULT_DEVICE_SERIAL;
      }
      // Default selectedDeviceSerials to only the currently selected device so the UI shows the default
      this.selectedDeviceSerials = [this.selectedDeviceSerial];
    },

    setSelectedDeviceSerial(serial: string) {
      this.selectedDeviceSerial = serial;
    },
    getSelectedDevice() {
      return (
        this.devices.find(
          (d) => (d.usbInfo?.serial || DEFAULT_DEVICE_SERIAL) === this.selectedDeviceSerial,
        ) ||
        this.devices.find((d) => !d.usbInfo?.serial) ||
        this.devices[0] ||
        null
      );
    },
  },
});
