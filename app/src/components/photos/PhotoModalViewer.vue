<template>
  <ModalDialog @close="close">
    <ImageViewer :filePath="path" />
  </ModalDialog>
</template>

<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import ImageViewer from '@/components/cirrus/viewers/ImageViewer.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'

const router = useRouter()
const route = useRoute()
const path = route.params.path as string

function close() {
  router.back()
}
</script>
