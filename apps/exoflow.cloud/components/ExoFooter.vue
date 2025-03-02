<template>
  <footer class="exo-footer">
    <div class="footer-content">
      <p class="copyright">
        © {{ currentYear }} ExoFlow. All rights reserved.
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
.exo-footer {
  width: 100%;
  padding: 1.5rem 0;
  background-color: #242424;
  margin-top: auto;
}

.footer-content {
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
  color: #888;
}

.footer-links {
  display: flex;
  gap: 1.5rem;
}

.footer-links a {
  color: #888;
  text-decoration: none;
  font-size: 0.9rem;
  transition: color 0.2s ease;
}

.footer-links a:hover {
  color: #e0e0e0;
}
</style>
