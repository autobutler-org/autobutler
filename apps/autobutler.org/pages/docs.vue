<template>
  <div class="documentation">
    <div class="docs-container">
      <h1>Documentation</h1>
      <div class="docs-content">
        <aside class="docs-sidebar">
          <nav>
            <ul>
              <li>
                <NuxtLink to="/docs/getting-started">Getting Started</NuxtLink>
              </li>
              <li><NuxtLink to="/docs/installation">Installation</NuxtLink></li>
              <li>
                <NuxtLink to="/docs/configuration">Configuration</NuxtLink>
              </li>
              <li>
                <NuxtLink to="/docs/api-reference">API Reference</NuxtLink>
              </li>
            </ul>
          </nav>
        </aside>
        <main class="docs-main">
          <slot />
        </main>
      </div>
    </div>
  </div>
</template>

<style scoped>
.documentation {
  flex: 1;
  padding: 2rem 0;
}

.docs-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
}

.docs-content {
  display: grid;
  grid-template-columns: 250px 1fr;
  gap: 2rem;
  margin-top: 2rem;
}

.docs-sidebar {
  border-right: 1px solid #333;
  padding-right: 1rem;
}

.docs-sidebar ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.docs-sidebar li {
  margin-bottom: 0.75rem;
}

.docs-sidebar a {
  color: #888;
  text-decoration: none;
  transition: color 0.2s ease;
}

.docs-sidebar a:hover {
  color: #e0e0e0;
}

.docs-main {
  min-height: 500px;
}
</style>
