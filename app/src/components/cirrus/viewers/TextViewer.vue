<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'

const props = defineProps<{
  filePath: string
}>()

const content = ref('')
const loading = ref(true)
const error = ref<string | null>(null)

const filename = computed(() => {
  const parts = props.filePath.split('/')
  return parts[parts.length - 1]
})

onMounted(async () => {
  try {
    const response = await fetch(`/api/v1/download/cirrus/${props.filePath}`)
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

<template>
  <div class="text-viewer-container">
    <div class="text-viewer-header">
      <span class="text-viewer-filename">{{ filename }}</span>
    </div>
    <div v-if="loading" class="text-viewer-loading">Loading...</div>
    <div v-else-if="error" class="text-viewer-error">{{ error }}</div>
    <pre v-else class="text-viewer-content">{{ content }}</pre>
  </div>
</template>

<style lang="scss" scoped>
.text-viewer-container {
  display: flex;
  flex-direction: column;
  width: 90vw;
  height: 80vh;
  background: var(--color-gray-900);
  border-radius: var(--border-radius-lg);
  overflow: hidden;

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-100);
  }
}

.text-viewer-header {
  display: flex;
  align-items: center;
  padding: var(--spacing-md) var(--spacing-lg);
  background: var(--color-gray-800);
  border-bottom: 1px solid var(--color-gray-700);

  @media (prefers-color-scheme: light) {
    background: var(--color-gray-200);
    border-bottom-color: var(--color-gray-300);
  }
}

.text-viewer-filename {
  font-family: monospace;
  font-size: var(--font-size-sm);
  color: var(--color-gray-300);

  @media (prefers-color-scheme: light) {
    color: var(--color-gray-700);
  }
}

.text-viewer-loading,
.text-viewer-error {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: var(--color-gray-400);
}

.text-viewer-error {
  color: var(--color-red-500);
}

.text-viewer-content {
  flex: 1;
  overflow: auto;
  padding: var(--spacing-lg);
  margin: 0;
  font-family: monospace;
  font-size: var(--font-size-sm);
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  color: var(--color-gray-200);

  @media (prefers-color-scheme: light) {
    color: var(--color-gray-800);
  }
}
</style>
