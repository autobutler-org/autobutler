import type { Device } from '@/types/device';
import { defineStore } from 'pinia';

export const useCirrusDeviceStore = defineStore('cirrusDevice', {
  state: () => ({
    selectedDeviceSerial: '' as string,
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
      // Default selectedDeviceSerials to all available devices (so listing shows all)
      this.selectedDeviceSerials = filtered.map((d) => d.usbInfo?.serial || '');
      // If the current selected device is not present, fallback to first internal
      if (
        !filtered.some((d) => d.usbInfo?.serial === this.selectedDeviceSerial)
      ) {
        const internal = filtered.find((d) => !d.usbInfo?.serial);
        this.selectedDeviceSerial = internal
          ? ''
          : filtered[0]?.usbInfo?.serial || '';
      }
    },
    setSelectedDeviceSerial(serial: string) {
      this.selectedDeviceSerial = serial;
    },
    getSelectedDevice() {
      return (
        this.devices.find(
          (d) => (d.usbInfo?.serial || '') === this.selectedDeviceSerial,
        ) ||
        this.devices.find((d) => !d.usbInfo?.serial) ||
        this.devices[0] ||
        null
      );
    },
  },
});
