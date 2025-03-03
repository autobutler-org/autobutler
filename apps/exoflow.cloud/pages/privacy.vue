<template>
  <div class="privacy-page">
    <h1>Privacy Policy</h1>
    <div class="content">
      <p>Last updated: {{ lastUpdated }}</p>
      <section v-for="(section, index) in sections" :key="index">
        <h2>{{ section.title }}</h2>
        <p>{{ section.content }}</p>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
const lastUpdated = new Date().toLocaleDateString();
const sections = [
  {
    title: "Information Collection",
    content:
      "We collect information to provide better services to our users...",
  },
  {
    title: "Data Usage",
    content: "Your data is used to improve our services and user experience...",
  },
];
</script>

<style scoped>
.content {
  max-width: 800px;
  margin: 0 auto;
}

section {
  margin: 2rem 0;
}
</style>
