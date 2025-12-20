<script setup lang="ts">
defineProps<{
  filePath: string
}>()
</script>

<template>
  <iframe
    class="pdf-viewer-frame"
    :src="`/api/v1/cirrus/${filePath}`"
    type="application/pdf"
  />
</template>

<style lang="scss" scoped>
.pdf-viewer-frame {
  width: 90vw;
  height: 90vh;
  border: none;
}
</style>
