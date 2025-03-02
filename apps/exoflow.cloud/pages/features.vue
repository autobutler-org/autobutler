<template>
  <div class="features-page">
    <h1>Features</h1>
    <div class="features-grid">
      <div
        class="feature-card"
        v-for="(feature, index) in features"
        :key="index"
      >
        <h3>{{ feature.title }}</h3>
        <p>{{ feature.description }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const features = [
  {
    title: "Multi-Cloud Deployment",
    description:
      "Deploy your applications across multiple cloud providers seamlessly.",
  },
  {
    title: "Automated Failover",
    description:
      "Automatic failover and recovery across regions and providers.",
  },
  {
    title: "Platform Agnostic",
    description: "Write once, deploy anywhere with our abstraction layer.",
  },
];
</script>

<style scoped>
.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
  margin-top: 2rem;
}

.feature-card {
  padding: 1.5rem;
  background: #242424;
  border-radius: 8px;
  border: 1px solid #333;
}
</style>
