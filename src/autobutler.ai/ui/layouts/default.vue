<template>
  <div class="layout">
    <ButlerHeader />
    <div class="page-container">
      <slot />
    </div>
    <ButlerFooter />
  </div>
</template>

<script setup lang="ts">
import ButlerFooter from "../components/ButlerFooter.vue";
import ButlerHeader from "../components/ButlerHeader.vue";
</script>

<style scoped>
.layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  position: relative;
  padding-top: 4rem; /* Add padding for fixed header */
}

.page-container {
  flex: 1;
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem 1rem;
  position: relative;
  z-index: 1;
}

/* Add responsive design */
@media (max-width: 768px) {
  .page-container {
    padding: 1rem;
  }
}
</style>

<style>
/* Global styles */
:root {
  --primary-color: #4caf50;
  --primary-light: #81c784;
  --text-color: #e0e0e0;
  --bg-color: #1a1a1a;
  --border-color: #333;
}

body {
  margin: 0;
  padding: 0;
  background-color: var(--bg-color);
  color: var(--text-color);
  font-family: Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

h1,
h2,
h3,
h4,
h5,
h6 {
  margin-top: 0;
  line-height: 1.2;
}

a {
  color: var(--primary-color);
  text-decoration: none;
  transition: color 0.2s ease;
}

a:hover {
  color: var(--primary-light);
}
</style>
