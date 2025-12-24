<script setup lang="ts">
import GradientOverlays from '@/components/home/GradientOverlays.vue'
import HeroSection from '@/components/home/HeroSection.vue'
import AppFooter from '@/components/home/AppFooter.vue'
</script>
<template>
  <div class="landing-body">
    <div class="landing-main site-fullscreen-col">
      <GradientOverlays />
      <div class="landing-container">
        <HeroSection />
      </div>
      <AppFooter />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.landing-body {
  margin: 0;
  padding: 0;
  min-height: 100vh;
  background: linear-gradient(180deg, hsl(225, 25%, 15%) 0%, hsl(225, 30%, 10%) 100%);

  @media (prefers-color-scheme: light) {
    background: linear-gradient(180deg, hsl(225, 15%, 95%) 0%, hsl(225, 20%, 98%) 100%);
  }
}

.landing-main {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  position: relative;
}

.site-fullscreen-col {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.landing-container {
  display: flex;
  flex-direction: column;
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
  padding: 0 $spacing-2xl;
}
</style>
