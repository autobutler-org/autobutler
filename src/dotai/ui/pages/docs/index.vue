<template>
  <div>
    <!-- This will redirect before rendering -->
  </div>
</template>

<script setup lang="ts">
// Makes sure the markdown pages are rendered before the generation is done
prerenderRoutes([
  "/docs/api-reference",
  "/docs/configuration",
  "/docs/examples",
  "/docs/getting-started",
  "/docs/help",
  "/docs/install",
  "/docs/quick-start",
  "/docs/welcome",
])
await navigateTo("/docs/welcome");
</script>
