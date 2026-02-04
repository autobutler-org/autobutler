<template>
  <iframe
    class="pdf-viewer-frame"
    :src="CirrusService.getDownloadUrl(filePath)"
    type="application/pdf"
  />
</template>

<script lang="ts" setup>
defineProps<{
  filePath: string;
}>();

import CirrusService from '@/services/cirrusService';
</script>

<style lang="scss" scoped>
.pdf-viewer-frame {
  width: 90vw;
  height: 90vh;
  border: none;
}
</style>
