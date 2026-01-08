export interface UsbInfo {
  path: string;
  vendorID: string;
  productID: string;
  manufacturer: string;
  product: string;
  serial: string;
  mountPath: string;
}

export interface Device {
  name: string;
  type: string;
  device_path: string;
  mount_point: string;
  file_system: string;
  total_bytes: number;
  used_bytes: number;
  avail_bytes: number;
  percent_used: number;
  is_internal: boolean;
  is_removable: boolean;
  is_read_only: boolean;
  model: string;
  categories: Record<string, number>;
  usb_info?: UsbInfo;
}
