import { defineStore } from 'pinia';

export interface DeviceNotificationState {
  lastViewedTimestamp: number;
  hasUnviewedDevices: boolean;
}

const STORAGE_KEY = 'autobutler_device_notifications';

export const useDeviceNotificationStore = defineStore('deviceNotifications', {
  state: (): DeviceNotificationState => {
    // Load from localStorage
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      try {
        return JSON.parse(stored);
      } catch (e) {
        console.error('Failed to parse device notification state:', e);
      }
    }
    return {
      lastViewedTimestamp: Date.now(),
      hasUnviewedDevices: false,
    };
  },
  actions: {
    markDevicesAsViewed() {
      this.lastViewedTimestamp = Date.now();
      this.hasUnviewedDevices = false;
      this.saveToStorage();
    },
    setHasUnviewedDevices(hasUnviewed: boolean) {
      this.hasUnviewedDevices = hasUnviewed;
      this.saveToStorage();
    },
    saveToStorage() {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          lastViewedTimestamp: this.lastViewedTimestamp,
          hasUnviewedDevices: this.hasUnviewedDevices,
        }),
      );
    },
  },
});
