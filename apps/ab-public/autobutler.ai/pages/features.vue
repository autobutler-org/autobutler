<template>
  <PageContainer>
    <h1>Features</h1>
    <div class="features-grid">
      <div v-for="feature in features" :key="feature.title" class="feature-card">
        <h3>{{ feature.title }}</h3>
        <p>{{ feature.description }}</p>
      </div>
    </div>
  </PageContainer>
</template>

<script setup lang="ts">
const features = [
  {
    title: "Automated Workflows",
    description: "Streamline your processes with intelligent automation"
  },
  {
    title: "Smart Integration",
    description: "Seamlessly connect with your existing tools and services"
  },
  {
    title: "Real-time Analytics",
    description: "Get instant insights into your operations"
  },
  // Add more features as needed
];
</script>

<style scoped>
h1 {
  font-size: 2.5rem;
  margin-bottom: 2rem;
  color: #fff;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
}

.feature-card {
  padding: 1.5rem;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  transition: all 0.3s ease;
}

.feature-card:hover {
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.1),
    rgba(0, 187, 255, 0.1)
  );
  border-color: rgba(0, 255, 170, 0.3);
}
</style> 