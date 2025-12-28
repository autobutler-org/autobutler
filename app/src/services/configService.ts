import { useConfigStore, type ConfigState } from '@/stores/config'

export default class ConfigService {
  static initAsync = async () => {
    const res = await fetch('/config.json')
    let data: ConfigState = {
      featureFlags: {},
    }
    if (res.ok) {
      data = (await res.json()) as ConfigState
    }
    const configStore = useConfigStore()
    configStore.setFeatureFlags(data.featureFlags || {})
  }
}
