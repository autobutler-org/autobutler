<template>
  <div class="pdf-viewer">
    <iframe v-if="src" :src="src" type="application/pdf" width="100%" height="100%"></iframe>
    <div v-else class="error-text">No PDF file specified.</div>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ src: string }>()
</script>

<style lang="scss" scoped>
.pdf-viewer {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  background: $color-gray-100;

  iframe {
    border: none;
    min-height: 80vh;
  }
}
</style>
