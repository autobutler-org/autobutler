<template>
  <div class="pdf-viewer">
    <iframe
      v-if="filePath"
      :src="`/api/v1/download/cirrus/${encodeURIComponent(filePath)}`"
      type="application/pdf"
      width="100%"
      height="100%"
      style="border: none; min-height: 80vh"
    ></iframe>
    <div v-else class="error-text">No PDF file specified.</div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ filePath: string }>()
</script>

<style scoped>
.pdf-viewer {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  background: var(--color-gray-100);
}
</style>
