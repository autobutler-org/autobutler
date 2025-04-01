<template>
  <div class="landing-page">
    <div class="gradient-overlay"></div>
    <div class="card">
      <h1>Your Private AI Home Assistant</h1>
      <p>
        Meet AutoButler, your self-hosted, privacy-focused home assistant powered by DeepSeek and Home Assistant. Unlike cloud-based alternatives, your data stays in your home, giving you complete control over your smart home experience.
      </p>
      <div class="features">
        <h2>Smart Features That Matter</h2>
        <ul>
          <li>Smart Grocery Management - Never run out of essentials</li>
          <li>Personalized Recipe Assistant - Get cooking inspiration</li>
          <li>Family Calendar Sync - Keep everyone in the loop</li>
          <li>Intelligent Lighting - Automatically adjusts based on your presence</li>
          <li>Enhanced Home Security - Monitor and protect what matters</li>
        </ul>
      </div>
      <p class="highlight">
        Your data, your control, your home. Experience the future of home automation without compromising your privacy.
      </p>
      <NuxtLink to="/about" class="btn">Get Started</NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
// Component logic can be added here if needed
</script>

<style>
@import '@/assets/css/common.css';

.features {
  margin: 2rem 0;
  padding: 1rem;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 8px;
}

.features ul {
  list-style: none;
  padding: 0;
}

.features li {
  margin: 0.8rem 0;
  font-size: 1.1rem;
}

.highlight {
  font-size: 1.2rem;
  font-weight: 500;
  margin: 2rem 0;
  padding: 1rem;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 8px;
  text-align: center;
}

.btn {
  display: inline-block;
  padding: 1rem 2rem;
  background: rgba(255, 255, 255, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 8px;
  color: white;
  text-decoration: none;
  font-weight: 500;
  transition: all 0.3s ease;
}

.btn:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: translateY(-2px);
}
</style> 