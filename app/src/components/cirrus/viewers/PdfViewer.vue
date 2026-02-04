<template>
  <iframe
    class="pdf-viewer-frame"
    :src="`/api/v1/cirrus/download/${filePath}`"
    type="application/pdf"
  />
</template>

<script lang="ts" setup>
defineProps<{
  filePath: string;
}>();
</script>

<style lang="scss" scoped>
.pdf-viewer-frame {
  width: 90vw;
  height: 90vh;
  border: none;
}
</style>
