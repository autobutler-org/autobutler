<template>
  <img class="file-viewer-media" :src="`/api/v1/download/cirrus/${filePath}`" :alt="filePath" />
</template>

<script setup lang="ts">
defineProps<{
  filePath: string
}>()
</script>

<style lang="scss" scoped>
.file-viewer-media {
  width: 100%;
  height: auto;
  object-fit: contain;
  max-height: 90vh;
  max-width: 90vw;
  margin-left: auto;
  margin-right: auto;
}
</style>
