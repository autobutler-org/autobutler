<template>
  <div class="terms-page">
    <h1>Terms of Service</h1>
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
    title: "Acceptance of Terms",
    content:
      "By accessing and using AutoButler, you accept and agree to be bound by these terms...",
  },
  {
    title: "Service Description",
    content:
      "AutoButler provides home automation and information cataloguing services...",
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
