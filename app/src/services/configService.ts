import { useConfigStore, type ConfigState } from '@/stores/config';
import HttpService from './httpService';

export default class ConfigService {
  static initAsync = async () => {
    let data: ConfigState = {
      featureFlags: {},
    };
    try {
      data = await HttpService.getAsJson<ConfigState>('/config.json');
    } catch {}
    const configStore = useConfigStore();
    configStore.setFeatureFlags(data.featureFlags || {});
  };
}
