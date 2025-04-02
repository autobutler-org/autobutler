<template>
  <div class="page-wrapper">
    <div class="gradient-overlay"></div>
    <div class="content-card">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.page-wrapper {
  width: 100%;
  min-height: 80vh;
  position: relative;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: 2rem 1rem;
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

.content-card {
  background: rgba(28, 32, 34, 0.95);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 2rem;
  width: 100%;
  max-width: 1200px;
}
</style> 