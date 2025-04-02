<template>
  <main class="terms-page">
    <div class="terms-content">
      <h1>Terms of Service</h1>
      <div class="section">
        <h2>1. Acceptance of Terms</h2>
        <p>By accessing and using AutoButler, you agree to be bound by these Terms of Service and all applicable laws and regulations.</p>
      </div>

      <div class="section">
        <h2>2. Use License</h2>
        <p>Permission is granted to temporarily access AutoButler for personal, non-commercial use only. This license does not include:</p>
        <ul>
          <li>Modifying or copying the materials</li>
          <li>Using the materials for commercial purposes</li>
          <li>Attempting to decompile or reverse engineer any software</li>
          <li>Removing any copyright or proprietary notations</li>
        </ul>
      </div>

      <div class="section">
        <h2>3. User Responsibilities</h2>
        <p>As a user of AutoButler, you agree to:</p>
        <ul>
          <li>Provide accurate and complete information</li>
          <li>Maintain the security of your account</li>
          <li>Use the service in compliance with all applicable laws</li>
        </ul>
      </div>

      <div class="section">
        <h2>4. Disclaimer</h2>
        <p>The materials on AutoButler are provided on an 'as is' basis. We make no warranties, expressed or implied, and hereby disclaim all other warranties including, without limitation, implied warranties or conditions of merchantability, fitness for a particular purpose, or non-infringement of intellectual property or other violation of rights.</p>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
// Component is stateless
</script>

<style scoped>
.terms-page {
  flex: 1;
  width: 100%;
  padding: 2rem 0;
}

.terms-content {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 1rem;
}

h1 {
  color: #4caf50;
  font-size: 2.5rem;
  margin-bottom: 2rem;
}

.section {
  margin-bottom: 2.5rem;
}

h2 {
  color: #81c784;
  font-size: 1.8rem;
  margin-bottom: 1rem;
}

p {
  color: #e0e0e0;
  line-height: 1.6;
  margin-bottom: 1rem;
}

ul {
  list-style-type: none;
  padding-left: 1.5rem;
}

ul li {
  color: #e0e0e0;
  line-height: 1.6;
  margin-bottom: 0.5rem;
  position: relative;
}

ul li::before {
  content: "•";
  color: #4caf50;
  position: absolute;
  left: -1.5rem;
  font-size: 1.2rem;
}
</style> 