<template>
  <main class="privacy-page">
    <div class="privacy-content">
      <h1>Privacy Policy</h1>
      <div class="section">
        <h2>1. Information We Collect</h2>
        <p>We collect information that you provide directly to us, including:</p>
        <ul>
          <li>Account information (name, email, password)</li>
          <li>Profile information</li>
          <li>Usage data and analytics</li>
        </ul>
      </div>

      <div class="section">
        <h2>2. How We Use Your Information</h2>
        <p>We use the collected information to:</p>
        <ul>
          <li>Provide and maintain our services</li>
          <li>Improve and personalize your experience</li>
          <li>Communicate with you about updates and support</li>
        </ul>
      </div>

      <div class="section">
        <h2>3. Data Security</h2>
        <p>We implement appropriate security measures to protect your personal information from unauthorized access, alteration, disclosure, or destruction.</p>
      </div>

      <div class="section">
        <h2>4. Contact Us</h2>
        <p>If you have any questions about this Privacy Policy, please contact us at:</p>
        <p>Email: privacy@autobutler.ai</p>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
// Component is stateless
</script>

<style scoped>
.privacy-page {
  flex: 1;
  width: 100%;
  padding: 2rem 0;
}

.privacy-content {
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