<template>
  <PageContainer>
    <div class="docs-layout">
      <aside class="sidebar">
        <nav>
          <ul>
            <li v-for="section in sections" :key="section">
              <a href="#" @click.prevent="currentSection = section">{{ section }}</a>
            </li>
          </ul>
        </nav>
      </aside>
      <main class="content">
        <h1>Documentation</h1>
        <p>Welcome to the AutoButler documentation. Select a topic from the sidebar to begin.</p>
      </main>
    </div>
  </PageContainer>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const sections = [
  'Getting Started',
  'Installation',
  'Configuration',
  'API Reference',
  'Examples'
];

const currentSection = ref('Getting Started');
</script>

<style scoped>
.docs-layout {
  display: grid;
  grid-template-columns: 250px 1fr;
  gap: 2rem;
}

.sidebar {
  border-right: 1px solid rgba(255, 255, 255, 0.1);
  padding-right: 2rem;
}

.sidebar ul {
  list-style: none;
  padding: 0;
}

.sidebar a {
  display: block;
  padding: 0.5rem 0;
  color: rgba(255, 255, 255, 0.8);
  text-decoration: none;
  transition: all 0.3s ease;
}

.sidebar a:hover {
  color: #fff;
  padding-left: 0.5rem;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.1),
    rgba(0, 187, 255, 0.1)
  );
}
</style> 