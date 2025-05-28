<template>
  <PageContainer>
    <div class="docs-index">
      <div class="docs-header">
        <h1>AutoButler Documentation</h1>
        <p>Welcome to the complete AutoButler documentation. Choose a topic below to get started.</p>
      </div>
      
      <div class="docs-grid">
        <NuxtLink 
          v-for="doc in sortedDocs" 
          :key="doc._path"
          :to="doc._path"
          class="doc-card"
        >
          <h3>{{ doc.navigation?.title || doc.title }}</h3>
          <p>{{ doc.description }}</p>
        </NuxtLink>
      </div>
    </div>
  </PageContainer>
</template>

<script setup>
// Fetch all documentation files using Nuxt Content composables
const { data: allDocs } = await queryContent('/docs').find()

// Computed properties
const sortedDocs = computed(() => 
  allDocs?.sort((a, b) => (a.navigation?.order || 999) - (b.navigation?.order || 999)) || []
)

// SEO
useSeoMeta({
  title: 'AutoButler Documentation',
  description: 'Complete documentation for AutoButler automation platform',
})
</script>

<style scoped>
.docs-index {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.docs-header {
  text-align: center;
  margin-bottom: 3rem;
}

.docs-header h1 {
  font-size: 3rem;
  font-weight: 700;
  margin-bottom: 1rem;
  background: linear-gradient(135deg, rgba(0, 255, 170, 0.9), rgba(0, 187, 255, 0.9));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.docs-header p {
  font-size: 1.25rem;
  color: rgba(255, 255, 255, 0.8);
  max-width: 600px;
  margin: 0 auto;
}

.docs-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
}

.doc-card {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.75rem;
  padding: 1.5rem;
  text-decoration: none;
  transition: all 0.3s ease;
  display: block;
}

.doc-card:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(0, 255, 170, 0.3);
  transform: translateY(-2px);
  box-shadow: 0 10px 25px rgba(0, 255, 170, 0.1);
}

.doc-card h3 {
  font-size: 1.375rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
  color: #fff;
}

.doc-card p {
  color: rgba(255, 255, 255, 0.7);
  line-height: 1.6;
  margin: 0;
}

@media (max-width: 768px) {
  .docs-index {
    padding: 1rem;
  }
  
  .docs-header h1 {
    font-size: 2.25rem;
  }
  
  .docs-header p {
    font-size: 1.125rem;
  }
  
  .docs-grid {
    grid-template-columns: 1fr;
    gap: 1rem;
  }
  
  .doc-card {
    padding: 1.25rem;
  }
}
</style> 