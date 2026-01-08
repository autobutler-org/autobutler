import type { Device } from '@/types/device';
import { defineStore } from 'pinia';

export const useCirrusDeviceStore = defineStore('cirrusDevice', {
  state: () => ({
    selectedDeviceSerial: '' as string,
    devices: [] as Device[],
  }),
  actions: {
    setDevices(devices: Device[]) {
      // Only include devices that are internal or have a mount_point (i.e., mounted)
      const filtered = devices.filter(
        (d) => !d.usb_info?.serial || !!d.mount_point,
      );
      this.devices = filtered;
      // If the current selected device is not present, fallback to first internal
      if (
        !filtered.some((d) => d.usb_info?.serial === this.selectedDeviceSerial)
      ) {
        const internal = filtered.find((d) => !d.usb_info?.serial);
        this.selectedDeviceSerial = internal
          ? ''
          : filtered[0]?.usb_info?.serial || '';
      }
    },
    setSelectedDeviceSerial(serial: string) {
      this.selectedDeviceSerial = serial;
    },
    getSelectedDevice() {
      return (
        this.devices.find(
          (d) => (d.usb_info?.serial || '') === this.selectedDeviceSerial,
        ) ||
        this.devices.find((d) => !d.usb_info?.serial) ||
        this.devices[0] ||
        null
      );
    },
  },
});
