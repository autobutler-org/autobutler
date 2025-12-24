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

<script setup lang="ts">
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
  color: $color-gray-300;
  font-size: $font-size-lg;

  @media (prefers-color-scheme: light) {
    color: $color-gray-700;
  }
}

.download-btn {
  color: white;
  background-color: $color-primary-600;
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
    background-color: $color-primary-700;
  }
}
</style>
