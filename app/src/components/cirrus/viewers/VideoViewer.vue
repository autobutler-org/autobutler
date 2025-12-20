<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  filePath: string
}>()

const videoType = computed(() => {
  const extension = props.filePath.split('.').pop()?.toLowerCase() || ''
  switch (extension) {
    case 'webm':
      return 'video/webm'
    case 'ogg':
    case 'ogv':
      return 'video/ogg'
    case 'mov':
      return 'video/quicktime'
    case 'avi':
      return 'video/avi'
    default:
      return 'video/mp4'
  }
})
</script>

<template>
  <video class="file-viewer-media" controls>
    <source :src="`/api/v1/download/cirrus/${filePath}`" :type="videoType" />
    Your browser does not support this video format.
  </video>
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
