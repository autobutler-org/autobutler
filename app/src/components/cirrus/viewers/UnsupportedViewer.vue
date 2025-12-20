<script setup lang="ts">
defineProps<{
  filePath: string
}>()
</script>

<template>
  <div class="unsupported-viewer">
    <div class="unsupported-viewer-message">
      <p>This file type cannot yet be viewed directly</p>
    </div>
    <div class="unsupported-viewer-actions">
      <a :href="`/api/v1/cirrus/${filePath}`" class="download-btn" download> Download file </a>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.unsupported-viewer {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-xl);
  padding: var(--spacing-3xl);
  text-align: center;
}

.unsupported-viewer-message {
  color: var(--color-gray-300);
  font-size: var(--font-size-lg);

  @media (prefers-color-scheme: light) {
    color: var(--color-gray-700);
  }
}

.download-btn {
  color: white;
  background-color: var(--color-primary-600);
  padding: var(--spacing-md) var(--spacing-2xl);
  border-radius: var(--border-radius-lg);
  transition: background-color 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-sm);
  text-decoration: none;
  border: none;
  cursor: pointer;

  &:hover {
    background-color: var(--color-primary-700);
  }
}
</style>
