<script setup lang="ts">
import type { ServiceIcon } from '@/types/home'

defineProps<{
  icons: ServiceIcon[]
}>()

function handleDisabledClick() {
  alert('Coming soon!')
}
</script>

<template>
  <div class="service-icons-container">
    <div class="service-icons">
      <template v-for="icon in icons" :key="icon.name">
        <a v-if="icon.enabled" :href="icon.href" class="service-icon-button">
          <div class="service-icon-bg">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              v-html="icon.iconSVG"
            />
          </div>
          <span class="service-icon-label">{{ icon.label }}</span>
        </a>
        <div
          v-else
          class="service-icon-button service-icon-button--disabled"
          @click="handleDisabledClick"
          style="cursor: pointer"
        >
          <div class="service-icon-bg">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              v-html="icon.iconSVG"
            />
          </div>
          <span class="service-icon-label">{{ icon.label }}</span>
        </div>
      </template>
    </div>
  </div>
</template>
