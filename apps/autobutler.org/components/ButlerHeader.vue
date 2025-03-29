<template>
  <header class="header">
    <div class="header-content">
      <div class="logo">
        <NuxtLink to="/" class="logo-link">AutoButler</NuxtLink>
      </div>
      <nav class="header-links">
        <NuxtLink to="/features">Features</NuxtLink>
        <NuxtLink to="/docs">Documentation</NuxtLink>
        <NuxtLink to="/pricing">Pricing</NuxtLink>
        <NuxtLink to="/login" class="login-link">Login</NuxtLink>
      </nav>
    </div>
  </header>
</template>

<script setup lang="ts">
// Header is currently stateless, but we can add logic here later if needed
</script>

<style scoped>
.header {
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
