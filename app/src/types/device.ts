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
  devicePath: string;
  mountPoint: string;
  fileSystem: string;
  totalBytes: number;
  usedBytes: number;
  availableBytes: number;
  isInternal: boolean;
  model: string;
  categories: Record<string, number>;
  usbInfo?: UsbInfo;
  isEnabled: boolean;
}
