import type { Device } from '@/types/device';
import HttpService from './httpService';

export interface DevicesStatusResponse {
  devices: Device[];
}

export interface UsbDevice {
  path: string;
  vendorID: string;
  productID: string;
  manufacturer: string;
  product: string;
  serial: string;
}

export interface UsbDevicesResponse {
  devices: Array<UsbDevice>;
  count: number;
}

export default class DevicesService {
  // TODO: Return error type and not just void
  static async disableUsbStorageDevice(serial: string): Promise<void> {
    await HttpService.delete(`/api/v1/storage/devices/usb/${serial}`);
  }

  // TODO: Return error type and not just void
  static async enableUsbStorageDevice(serial: string): Promise<void> {
    await HttpService.post(`/api/v1/storage/devices/usb/${serial}`);
  }

  static async fetchDevicesStatus(): Promise<DevicesStatusResponse> {
    return await HttpService.getAsJson<DevicesStatusResponse>(
      '/api/v1/storage/devices/status',
    );
  }

  static async listUsbStorageDevices(): Promise<UsbDevicesResponse> {
    return await HttpService.getAsJson<UsbDevicesResponse>(
      '/api/v1/storage/devices/usb',
    );
  }
}
