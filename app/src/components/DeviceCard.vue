<template>
  <div class="device-card" @click="goToCirrus">
    <div class="device-card-header">
      <div class="device-card-icon">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
          <line x1="8" y1="21" x2="16" y2="21"></line>
          <line x1="12" y1="17" x2="12" y2="21"></line>
        </svg>
      </div>
      <div class="device-card-title-section">
        <h3 class="device-card-title">{{ device.name }}</h3>
        <p class="device-card-type">{{ device.type }} • {{ device.device_path }}</p>
      </div>
    </div>
    <div class="device-card-body">
      <StoragePartition :device="device" />
    </div>
    <div class="device-card-footer">
      <div class="device-card-mount">
        <span class="device-mount-label">Mount: {{ device.mount_point }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from 'vue';
import StoragePartition from './StoragePartition.vue';

const props = defineProps({
  device: {
    type: Object,
    required: true,
  },
});

function goToCirrus() {
  window.location.href = '/cirrus';
}
</script>

<style scoped>
</style>
