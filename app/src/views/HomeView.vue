<script setup lang="ts">
import GradientOverlays from '@/components/home/GradientOverlays.vue'
import TopNav from '@/components/home/TopNav.vue'
import HeroSection from '@/components/home/HeroSection.vue'
import AppFooter from '@/components/home/AppFooter.vue'
</script>

<template>
  <div class="landing-body">
    <main class="landing-main site-fullscreen-col">
      <GradientOverlays />
      <div class="landing-container">
        <TopNav />
        <HeroSection />
      </div>
      <AppFooter />
    </main>
  </div>
</template>
