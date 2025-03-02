<template>
  <main class="home">
    <div class="home-content">
      <h1>Welcome to ExoFlow</h1>
      <p class="intro">
        ExoFlow is an abstract set of templates and automation that allows for
        packaging a single codebase to run on multiple clouds. The goal is to
        simplify multi-cloud, multi-region failovers and deployments, enabling
        seamless and reliable response.
      </p>
      <h2>Key Features</h2>
      <ul class="features-list">
        <li>
          Write platform-agnostic code that translates to the host platform
        </li>
        <li>
          Define applications as networked microclouds using a single manifest
        </li>
        <li>
          Orchestrate deployments across selected providers (AWS, Azure, etc.)
        </li>
        <li>
          Automatic failover and redeployment to another host if a microcloud
          fails
        </li>
        <li>
          Proxy-based API translation for seamless integration with cloud
          services
        </li>
      </ul>
    </div>
  </main>
</template>

<script setup lang="ts">
// Component is currently stateless
</script>

<style scoped>
.home {
  flex: 1;
  width: 100%;
  padding: 2rem 0;
}

.home-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
}

h1 {
  color: #4caf50;
  font-size: 2.5rem;
  margin-bottom: 1.5rem;
}

.intro {
  font-size: 1.1rem;
  line-height: 1.6;
  color: #e0e0e0;
  margin-bottom: 2rem;
}

h2 {
  color: #81c784;
  font-size: 1.8rem;
  margin: 2rem 0 1.5rem;
}

.features-list {
  list-style-type: none;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.features-list li {
  position: relative;
  padding-left: 1.5rem;
  color: #e0e0e0;
  line-height: 1.5;
}

.features-list li::before {
  content: "•";
  color: #4caf50;
  position: absolute;
  left: 0;
  font-size: 1.2rem;
}
</style>
