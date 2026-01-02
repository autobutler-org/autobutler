import { defineStore } from 'pinia';

export interface FeatureFlags {
  [key: string]: boolean;
}

export interface ConfigState {
  featureFlags: FeatureFlags;
}

export const useConfigStore = defineStore('config', {
  state: (): ConfigState => ({
    featureFlags: {},
  }),
  actions: {
    setFeatureFlags(flags: FeatureFlags) {
      this.featureFlags = flags;
    },
  },
});
