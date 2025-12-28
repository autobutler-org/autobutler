import { useConfigStore } from '@/stores/config'
import { storeToRefs } from 'pinia'

export const useFeatureFlag = (flag: string): boolean => {
  const configStore = useConfigStore()
  const { featureFlags } = storeToRefs(configStore)
  return !!featureFlags.value[flag]
}
