<template>
  <header class="exo-header">
    <div class="header-content">
      <div class="logo">
        <a href="/" class="logo-link">ExoFlow</a>
      </div>
      <nav class="header-links">
        <a href="/features">Features</a>
        <a href="/docs">Documentation</a>
        <a href="/pricing">Pricing</a>
        <a href="/login" class="login-link">Login</a>
      </nav>
    </div>
  </header>
</template>

<script setup lang="ts">
// Header is currently stateless, but we can add logic here later if needed
</script>

<style scoped>
.exo-header {
  width: 100%;
  padding: 1rem 0;
  background-color: #242424;
  border-bottom: 1px solid #333;
}

.header-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logo-link {
  color: #e0e0e0;
  text-decoration: none;
  font-size: 1.5rem;
  font-weight: bold;
  transition: color 0.2s ease;
}

.logo-link:hover {
  color: #fff;
}

.header-links {
  display: flex;
  gap: 2rem;
  align-items: center;
}

.header-links a {
  color: #888;
  text-decoration: none;
  font-size: 0.95rem;
  transition: color 0.2s ease;
}

.header-links a:hover {
  color: #e0e0e0;
}

.login-link {
  padding: 0.5rem 1rem;
  border: 1px solid #888;
  border-radius: 4px;
}

.login-link:hover {
  border-color: #e0e0e0;
}
</style>
