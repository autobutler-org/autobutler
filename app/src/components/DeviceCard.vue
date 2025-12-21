<template>
  <div class="device-card" @click="goToCirrus">
    <div class="device-card-header">
      <div class="device-card-icon">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
          <line x1="8" y1="21" x2="16" y2="21"></line>
          <line x1="12" y1="17" x2="12" y2="21"></line>
        </svg>
      </div>
      <div class="device-card-title-section">
        <h3 class="device-card-title">{{ device.name }}</h3>
        <p class="device-card-type">
          {{ device.is_internal ? 'External' : 'Internal' }} • {{ device.file_system }} •
          {{ device.device_path }}
        </p>
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
import StoragePartition from './StoragePartition.vue'

defineProps({
  device: {
    type: Object,
    required: true,
  },
})

function goToCirrus() {
  window.location.href = '/cirrus'
}
</script>

<style lang="scss" scoped>
.device-card {
  background: rgba(0, 0, 0, 0.12);
  border-radius: 18px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  padding: 2.2rem 2.2rem 1.5rem 2.2rem;
  display: flex;
  flex-direction: column;
  gap: 1.2rem;
  max-width: 600px;
  margin: 0 auto;
}
.device-card-header {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}
.device-card-icon {
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.device-card-icon svg {
  width: 48px;
  height: 48px;
  color: white;
}
.device-card-title-section {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}
.device-card-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: white;
  margin: 0;
}
.device-card-type {
  font-size: 1rem;
  color: #b3b3b3;
  margin: 0;
}
.device-card-body {
  margin-top: 0.5rem;
}
.device-card-footer {
  margin-top: 0.8rem;
}
.device-card-mount {
  font-size: 1rem;
  color: #e0e0e0;
}
</style>
