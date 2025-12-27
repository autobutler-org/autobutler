<template>
  <div class="text-viewer-container">
    <div class="text-viewer-header">
      <span class="text-viewer-filename">{{ filename }}</span>
    </div>
    <div v-if="loading" class="text-viewer-loading">Loading...</div>
    <div v-else-if="error" class="text-viewer-error">
      {{ error }}
    </div>
    <pre v-else class="text-viewer-content">{{ content }}</pre>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue'

const props = defineProps<{
  src: string
}>()

const content = ref('')
const loading = ref(true)
const error = ref<string | null>(null)

const filename = computed(() => props.src.split('/').pop())

onMounted(async () => {
  try {
    const response = await fetch(props.src)
    if (!response.ok) {
      throw new Error('Failed to load file')
    }
    content.value = await response.text()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load file'
  } finally {
    loading.value = false
  }
})
</script>

<style lang="scss" scoped>
.text-viewer-container {
  display: flex;
  flex-direction: column;
  width: 90vw;
  height: 80vh;
  background: $color-gray-900;
  border-radius: $border-radius-lg;
  overflow: hidden;

  @media (prefers-color-scheme: light) {
    background: $color-gray-100;
  }
}

.text-viewer-header {
  display: flex;
  align-items: center;
  padding: $spacing-md $spacing-lg;
  background: $color-gray-800;
  border-bottom: 1px solid $color-gray-700;

  @media (prefers-color-scheme: light) {
    background: $color-gray-200;
    border-bottom-color: $color-gray-300;
  }
}

.text-viewer-filename {
  font-family: monospace;
  font-size: $font-size-sm;
  color: $color-gray-300;

  @media (prefers-color-scheme: light) {
    color: $color-gray-700;
  }
}

.text-viewer-loading,
.text-viewer-error {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: $color-gray-400;
}

.text-viewer-error {
  color: $color-red-500;
}

.text-viewer-content {
  flex: 1;
  overflow: auto;
  padding: $spacing-lg;
  margin: 0;
  font-family: monospace;
  font-size: $font-size-sm;
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  color: $color-gray-200;

  @media (prefers-color-scheme: light) {
    color: $color-gray-800;
  }
}
</style>
