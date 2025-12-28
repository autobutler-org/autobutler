<script lang="ts" setup>
import AppFooter from '@/components/home/AppFooter.vue'
import GradientOverlays from '@/components/home/GradientOverlays.vue'
import HeroSection from '@/components/home/HeroSection.vue'
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
  background: linear-gradient(180deg, $color-gray-900 0%, $color-gray-950 100%);

  @media (prefers-color-scheme: light) {
    background: linear-gradient(
      180deg,
      $color-gray-100 0%,
      $color-gray-50 100%
    );
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
