import type { Device } from '@/types/device';
import HttpService from './httpService';

export interface DevicesStatusResponse {
  devices: Device[];
}

export default class DevicesService {
  static async fetchDevicesStatus(): Promise<DevicesStatusResponse> {
    return await HttpService.getAsJson<DevicesStatusResponse>(
      '/api/v1/storage/devices/status',
    );
  }
}
