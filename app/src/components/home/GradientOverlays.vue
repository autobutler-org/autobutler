<template>
  <div class="gradient-overlays">
    <div class="gradient-overlay gradient-overlay--top" />
    <div class="gradient-overlay gradient-overlay--bottom" />
  </div>
</template>

<style lang="scss" scoped>
.gradient-overlay {
  position: absolute;
  left: 0;
  right: 0;
  height: 30%;

  &--top {
    top: 0;
    background: linear-gradient(
      180deg,
      rgba(59, 130, 246, 0.1) 0%,
      transparent 100%
    );
  }

  &--bottom {
    bottom: 0;
    background: linear-gradient(
      0deg,
      rgba(139, 92, 246, 0.1) 0%,
      transparent 100%
    );
  }
}

.gradient-overlays {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
}
</style>
