<template>
  <div class="view-content site-fullscreen-col">
    <GradientOverlays />
    <div class="landing-container">
      <HeroSection />
    </div>
  </div>
</template>

<script lang="ts" setup>
import GradientOverlays from '@/components/home/GradientOverlays.vue';
import HeroSection from '@/components/home/HeroSection.vue';
</script>

<style lang="scss" scoped>
.view-content {
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
  flex: 1;
  justify-content: center;
  align-items: center;
  // max-width: 1400px;
  margin: 0 auto;
  width: 100%;
  padding: 0 $spacing-2xl;
  padding-top: 5rem;
  padding-bottom: $spacing-2xl;
}
</style>
