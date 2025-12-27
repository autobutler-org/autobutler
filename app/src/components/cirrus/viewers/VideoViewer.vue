<template>
  <video class="file-viewer-media" controls>
    <source :src="src" :type="mimetype" />
    Your browser does not support this video format.
  </video>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{
  src: string
}>()

// TODO: Move this logic to a utility module
const getVideoMimetypeFromExtension = (extension: string) => {
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
}

const mimetype = computed(() =>
  getVideoMimetypeFromExtension(props.src.split('.').pop()?.toLowerCase() || ''),
)
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
