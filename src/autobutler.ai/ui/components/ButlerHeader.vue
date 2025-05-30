<template>
  <header class="header">
    <div class="gradient-overlay"></div>
    <div class="header-content">
      <div class="logo">
        <NuxtLink to="/" class="logo-link">AutoButler</NuxtLink>
      </div>
      
      <!-- Desktop Navigation -->
      <nav class="header-links desktop-nav">
        <NuxtLink to="/features">Features</NuxtLink>
        <NuxtLink to="/docs">Documentation</NuxtLink>
        <NuxtLink to="/pricing">Pricing</NuxtLink>
        <NuxtLink to="/login" class="login-link">Login</NuxtLink>
      </nav>

      <!-- Mobile Hamburger Button -->
      <button 
        class="mobile-menu-button"
        @click="toggleMobileMenu"
        :aria-expanded="isMobileMenuOpen"
        aria-label="Toggle navigation menu"
      >
        <span class="hamburger-line" :class="{ 'open': isMobileMenuOpen }"></span>
        <span class="hamburger-line" :class="{ 'open': isMobileMenuOpen }"></span>
        <span class="hamburger-line" :class="{ 'open': isMobileMenuOpen }"></span>
      </button>
    </div>

    <!-- Mobile Navigation Menu -->
    <div class="mobile-nav" :class="{ 'open': isMobileMenuOpen }">
      <div class="mobile-nav-header">
        <button 
          class="mobile-close-button"
          @click="closeMobileMenu"
          aria-label="Close navigation menu"
        >
          <span class="close-icon">×</span>
        </button>
      </div>
      <nav class="mobile-nav-content">
        <NuxtLink to="/features" @click="closeMobileMenu">Features</NuxtLink>
        <NuxtLink to="/docs" @click="closeMobileMenu">Documentation</NuxtLink>
        <NuxtLink to="/pricing" @click="closeMobileMenu">Pricing</NuxtLink>
        <NuxtLink to="/login" class="mobile-login-link" @click="closeMobileMenu">Login</NuxtLink>
      </nav>
    </div>

    <!-- Mobile Menu Overlay -->
    <div 
      class="mobile-overlay" 
      :class="{ 'open': isMobileMenuOpen }"
      @click="closeMobileMenu"
    ></div>
  </header>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const isMobileMenuOpen = ref(false)

const toggleMobileMenu = () => {
  isMobileMenuOpen.value = !isMobileMenuOpen.value
}

const closeMobileMenu = () => {
  isMobileMenuOpen.value = false
}

// Close mobile menu when clicking outside or on escape key
onMounted(() => {
  const handleEscape = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      closeMobileMenu()
    }
  }
  
  document.addEventListener('keydown', handleEscape)
  
  onUnmounted(() => {
    document.removeEventListener('keydown', handleEscape)
  })
})
</script>

<style scoped>
.header {
  width: 100%;
  padding: 1rem 0;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 999;
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

/* Desktop Navigation */
.desktop-nav {
  display: flex;
  gap: 2rem;
  align-items: center;
}

.desktop-nav a {
  color: rgba(255, 255, 255, 0.8);
  text-decoration: none;
  font-size: 0.95rem;
  transition: all 0.3s ease;
  padding: 0.5rem 1rem;
  border-radius: 4px;
}

.desktop-nav a:hover {
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

/* Mobile Menu Button */
.mobile-menu-button {
  display: none;
  flex-direction: column;
  justify-content: space-around;
  width: 24px;
  height: 24px;
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 0;
  z-index: 102;
}

.hamburger-line {
  width: 100%;
  height: 2px;
  background: rgba(255, 255, 255, 0.8);
  transition: all 0.3s ease;
  transform-origin: center;
}

.hamburger-line.open:nth-child(1) {
  transform: rotate(45deg) translate(5px, 5px);
}

.hamburger-line.open:nth-child(2) {
  opacity: 0;
}

.hamburger-line.open:nth-child(3) {
  transform: rotate(-45deg) translate(7px, -6px);
}

/* Mobile Navigation */
.mobile-nav {
  position: fixed;
  top: 0;
  right: -300px;
  width: 300px;
  height: 100vh;
  background: rgba(28, 32, 34, 0.98);
  backdrop-filter: blur(20px);
  transition: right 0.3s ease;
  z-index: 1001;
  border-left: 1px solid rgba(255, 255, 255, 0.08);
}

.mobile-nav.open {
  right: 0;
}

.mobile-nav-header {
  padding: 1rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

.mobile-close-button {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.3s ease;
}

.close-icon {
  font-size: 1.5rem;
  color: rgba(255, 255, 255, 0.8);
  font-weight: 300;
  line-height: 1;
}

.mobile-close-button:hover {
  background: rgba(255, 255, 255, 0.1);
}

.mobile-close-button:hover .close-icon {
  color: #fff;
}

.mobile-nav-content {
  display: flex;
  flex-direction: column;
  padding: 2rem;
  gap: 1.5rem;
}

.mobile-nav-content a {
  color: rgba(255, 255, 255, 0.8);
  text-decoration: none;
  font-size: 1.1rem;
  padding: 1rem 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  transition: all 0.3s ease;
}

.mobile-nav-content a:hover,
.mobile-nav-content a:focus {
  color: #fff;
  padding-left: 0.5rem;
}

.mobile-login-link {
  border: 1px solid rgba(255, 255, 255, 0.15) !important;
  border-radius: 4px !important;
  padding: 1rem !important;
  text-align: center;
  margin-top: 1rem;
}

.mobile-login-link:hover {
  border-color: rgba(0, 255, 170, 0.3) !important;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.15),
    rgba(0, 187, 255, 0.15)
  ) !important;
  padding-left: 1rem !important;
}

/* Mobile Overlay */
.mobile-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.5);
  opacity: 0;
  visibility: hidden;
  transition: all 0.3s ease;
  z-index: 1000;
  pointer-events: none;
}

.mobile-overlay.open {
  opacity: 1;
  visibility: visible;
  pointer-events: auto;
}

/* Responsive Design */
@media (max-width: 768px) {
  .desktop-nav {
    display: none;
  }
  
  .mobile-menu-button {
    display: flex;
  }
}

@media (max-width: 480px) {
  .mobile-nav {
    width: 280px;
    right: -280px;
  }
  
  .header-content {
    padding: 0 1rem;
  }
  
  .logo-link {
    font-size: 1.3rem;
  }
}
</style>
