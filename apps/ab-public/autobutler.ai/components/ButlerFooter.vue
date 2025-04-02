<template>
  <footer class="footer">
    <div class="gradient-overlay"></div>
    <div class="footer-content">
      <p class="copyright">
        © {{ currentYear }} AutoButler. All rights reserved.
      </p>
      <div class="footer-links">
        <a href="/privacy">Privacy</a>
        <a href="/terms">Terms</a>
        <a href="/contact">Contact</a>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from "vue";

const currentYear = computed(() => new Date().getFullYear());
</script>

<style scoped>
.footer {
  width: 100%;
  padding: 1.5rem 0;
  position: relative;
  background: rgba(28, 32, 34, 0.95);
  backdrop-filter: blur(12px);
  margin-top: auto;
  border: 1px solid rgba(255, 255, 255, 0.08);
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
}

.footer-content {
  position: relative;
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.copyright {
  margin: 0;
  font-size: 0.9rem;
  color: rgba(255, 255, 255, 0.7);
}

.footer-links {
  display: flex;
  gap: 1.5rem;
}

.footer-links a {
  color: rgba(255, 255, 255, 0.7);
  text-decoration: none;
  font-size: 0.9rem;
  transition: all 0.3s ease;
  padding: 0.5rem 1rem;
  border-radius: 4px;
}

.footer-links a:hover {
  color: #fff;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.1),
    rgba(0, 187, 255, 0.1)
  );
}
</style>