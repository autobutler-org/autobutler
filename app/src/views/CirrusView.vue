<script setup lang="ts">
import CirrusFileExplorer from '@/components/cirrus/CirrusFileExplorer.vue'
</script>

<template>
  <main class="cirrus-view">
    <CirrusFileExplorer />
  </main>
</template>

<style scoped>
.cirrus-view {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
