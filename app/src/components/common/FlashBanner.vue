<template>
  <transition name="flash-banner-fade">
    <div v-if="show" class="flash-banner">
      <slot />
    </div>
  </transition>
</template>

<script lang="ts" setup>
import { onUnmounted, ref, watch } from 'vue'

const props = defineProps<{
  show: boolean
  duration?: number
}>()

const emit = defineEmits(['hide'])

const show = ref(props.show)
let bannerTimeout: ReturnType<typeof setTimeout> | null = null

watch(
  () => props.show,
  (val) => {
    show.value = val
    if (val && props.duration !== 0) {
      if (bannerTimeout) clearTimeout(bannerTimeout)
      bannerTimeout = setTimeout(() => {
        show.value = false
        emit('hide')
      }, props.duration ?? 1200)
    }
  },
  { immediate: true },
)

onUnmounted(() => {
  if (bannerTimeout) clearTimeout(bannerTimeout)
})
</script>

<style lang="scss" scoped>
.flash-banner {
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  background: $color-primary-600;
  color: $color-gray-50;
  padding: 0.5rem 2rem;
  border-radius: 0 0 1rem 1rem;
  font-weight: 600;
  font-size: $theme-font-size-base;
  box-shadow: 0 2px 8px rgba($color-gray-950, 0.15);
  pointer-events: none;
  z-index: 200;
  opacity: 0.95;
  animation: flash-banner-pop 0.3s;
}

@keyframes flash-banner-pop {
  0% {
    transform: translateX(-50%) scale(0.9);
    opacity: 0.5;
  }
  100% {
    transform: translateX(-50%) scale(1);
    opacity: 0.95;
  }
}

.flash-banner-fade-enter-active,
.flash-banner-fade-leave-active {
  transition: opacity 0.4s;
}
.flash-banner-fade-enter-from,
.flash-banner-fade-leave-to {
  opacity: 0;
}
.flash-banner-fade-enter-to,
.flash-banner-fade-leave-from {
  opacity: 0.95;
}
</style>
