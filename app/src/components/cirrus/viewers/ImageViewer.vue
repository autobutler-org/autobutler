<script setup lang="ts">
defineProps<{
  filePath: string
}>()
</script>

<template>
  <img
    class="file-viewer-media"
    :src="`/api/v1/cirrus/${filePath}`"
    :alt="filePath"
  />
</template>

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
