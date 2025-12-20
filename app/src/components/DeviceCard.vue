
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
        <p class="device-card-type">{{ device.is_internal ? 'External' : 'Internal' }} • {{ device.file_system }} • {{ device.device_path }}</p>
      </div>
    </div>
    <div class="device-card-body">
      <div class="storage-partition-component">
        <div class="storage-partition-info">
          <span class="storage-partition-label">Capacity: {{ (device.total_bytes / 1e9).toFixed(1) }} GB</span>
          <span class="storage-partition-used">{{ device.percent_used }}% used • {{ (device.used_bytes / 1e9).toFixed(1) }} GB</span>
        </div>
        <div class="storage-partition-bar">
          <div
            v-for="cat in partitionSegments(device)"
            :key="cat.key"
            class="storage-partition-segment"
            :class="cat.class"
            :style="{ width: cat.percent + '%'}"
            :title="cat.label + ': ' + cat.sizeDisplay"
          ></div>
        </div>
        <div class="storage-partition-legend">
          <div v-for="cat in partitionSegments(device)" :key="cat.key" class="storage-partition-legend-item">
            <span class="storage-partition-dot" :class="cat.class"></span>
            <span class="storage-partition-legend-label">{{ cat.label }} {{ cat.sizeDisplay }}</span>
          </div>
        </div>
      </div>
    </div>
    <div class="device-card-footer">
      <div class="device-card-mount">
        <span class="device-mount-label">Mount: {{ device.mount_point }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

defineProps({
  device: {
    type: Object,
    required: true,
  },
})

function goToCirrus() {
  window.location.href = '/cirrus'
}

function partitionSegments(device: any) {
  // Map categories to color classes and labels
  const map = [
    { key: 'system', class: 'storage-partition-system', label: 'System' },
    { key: 'documents', class: 'storage-partition-documents', label: 'Documents' },
    { key: 'media', class: 'storage-partition-media', label: 'Media' },
    { key: 'other', class: 'storage-partition-other', label: 'Other' },
    { key: 'free', class: 'storage-partition-free', label: 'Free' },
  ];
  // Get total for percentage calculation
  const total = (device.used_bytes + device.avail_bytes) / 1e9;
  // Compose segments
  return map.map(seg => {
    let size = 0;
    if (seg.key === 'free') {
      size = device.avail_bytes / 1e9;
    } else if (device.categories && device.categories[seg.label]) {
      size = device.categories[seg.label] || 0;
    }
    return {
      key: seg.key,
      class: seg.class,
      label: seg.label,
      percent: total ? (size / total * 100) : 0,
      sizeDisplay: size.toFixed(2) + ' GB',
    };
  }).filter(seg => seg.percent > 0);
}
</script>

<style lang="scss" scoped>
/* Add styles for device card and storage partition bar here if needed */
</style>
