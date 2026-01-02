import type { Component } from 'vue';

export interface ServiceIcon {
  name: string;
  label: string;
  href: string;
  component: Component;
  enabled: boolean;
}
