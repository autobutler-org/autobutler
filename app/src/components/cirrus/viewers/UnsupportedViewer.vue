<template>
  <div class="unsupported-viewer">
    <div class="unsupported-viewer-message">
      <p>This file type cannot yet be viewed directly</p>
    </div>
    <div class="unsupported-viewer-actions">
      <a :href="src" class="download-btn" download> Download file </a>
    </div>
  </div>
</template>

<script lang="ts" setup>
defineProps<{
  src: string
}>()
</script>

<style lang="scss" scoped>
.unsupported-viewer {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: $spacing-xl;
  padding: $spacing-3xl;
  text-align: center;
}

.unsupported-viewer-message {
  color: $theme-palette-text-secondary;
  font-size: $theme-font-size-lg;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-inverse;
  }
}

.download-btn {
  color: $theme-palette-text-inverse;
  background-color: $theme-palette-accent;
  padding: $spacing-md $spacing-2xl;
  border-radius: $border-radius-lg;
  transition: background-color 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: $spacing-sm;
  text-decoration: none;
  border: none;
  cursor: pointer;

  &:hover {
    background-color: $theme-palette-accent-hover;
  }
}
</style>
