<template>
  <header class="header">
    <div class="gradient-overlay"></div>
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
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: rgba(28, 32, 34, 0.95);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.gradient-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.02),
    rgba(0, 187, 255, 0.02)
  );
  pointer-events: none;
  z-index: -1;
}

.header-content {
  position: relative;
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  z-index: 1;
}

.logo-link {
  color: rgba(255, 255, 255, 0.95);
  text-decoration: none;
  font-size: 1.5rem;
  font-weight: bold;
  transition: all 0.3s ease;
}

.logo-link:hover {
  color: #fff;
  text-shadow: 0 0 15px rgba(0, 255, 170, 0.3);
}

.header-links {
  display: flex;
  gap: 2rem;
  align-items: center;
}

.header-links a {
  color: rgba(255, 255, 255, 0.8);
  text-decoration: none;
  font-size: 0.95rem;
  transition: all 0.3s ease;
  padding: 0.5rem 1rem;
  border-radius: 4px;
}

.header-links a:hover {
  color: #fff;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.1),
    rgba(0, 187, 255, 0.1)
  );
}

.login-link {
  border: 1px solid rgba(255, 255, 255, 0.15);
}

.login-link:hover {
  border-color: rgba(0, 255, 170, 0.3);
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.15),
    rgba(0, 187, 255, 0.15)
  );
}
</style>
