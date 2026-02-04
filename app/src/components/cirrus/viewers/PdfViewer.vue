<template>
  <iframe class="pdf-viewer-frame" :src="fileSrc" type="application/pdf" />
</template>

<script lang="ts" setup>
defineProps<{
  filePath: string;
}>();

const fileSrc = `/api/v1/cirrus/download?${new URLSearchParams({ filePath })}`;
</script>

<style lang="scss" scoped>
.pdf-viewer-frame {
  width: 90vw;
  height: 90vh;
  border: none;
}
</style>
