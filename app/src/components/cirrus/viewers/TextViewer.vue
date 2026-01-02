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
import TextFileService from '@/services/textFileService';
import { computed, onMounted, ref } from 'vue';

const props = defineProps<{
  src: string;
}>();

const content = ref('');
const loading = ref(true);
const error = ref<string | null>(null);

const filename = computed(() => props.src.split('/').pop());

onMounted(async () => {
  try {
    content.value = await TextFileService.fetchTextFile(props.src);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load file';
  } finally {
    loading.value = false;
  }
});
</script>

<style lang="scss" scoped>
.text-viewer-container {
  display: flex;
  flex-direction: column;
  width: 90vw;
  height: 80vh;
  background: $theme-palette-bg-nav;
  border-radius: $border-radius-lg;
  overflow: hidden;

  @media (prefers-color-scheme: light) {
    background: $theme-palette-bg-inverse;
  }
}

.text-viewer-header {
  display: flex;
  align-items: center;
  padding: $spacing-md $spacing-lg;
  background: $theme-palette-bg-secondary;
  border-bottom: 1px solid $theme-palette-border;

  @media (prefers-color-scheme: light) {
    background: $theme-palette-bg-inverse;
    border-bottom-color: $theme-palette-border-strong;
  }
}

.text-viewer-filename {
  font-family: monospace;
  font-size: $theme-font-size-sm;
  color: $theme-palette-text-secondary;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-inverse;
  }
}

.text-viewer-loading,
.text-viewer-error {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: $theme-palette-text-muted;
}

.text-viewer-error {
  color: $theme-palette-danger;
}

.text-viewer-content {
  flex: 1;
  overflow: auto;
  padding: $spacing-lg;
  margin: 0;
  font-family: monospace;
  font-size: $theme-font-size-sm;
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  color: $theme-palette-text-primary;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-inverse;
  }
}
</style>
