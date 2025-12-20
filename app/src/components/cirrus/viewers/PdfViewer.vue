<template>
  <iframe
    class="pdf-viewer-frame"
    :src="`/api/v1/download/cirrus/${filePath}`"
    type="application/pdf"
  />
</template>

<script setup lang="ts">
defineProps<{
  filePath: string
}>()
</script>

<style lang="scss" scoped>
.pdf-viewer-frame {
  width: 90vw;
  height: 90vh;
  border: none;
}
</style>
