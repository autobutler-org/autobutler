
<template>
  <div class="storage-partition-component">
    <div class="storage-partition-info">
      <span class="storage-partition-label">Capacity: {{ (device.total_bytes / 1_073_741_824).toFixed(1) }} GB</span>
      <span class="storage-partition-used">{{ device.percent_used }}% used • {{ (device.used_bytes / 1_073_741_824).toFixed(1) }} GB</span>
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
</template>

<script setup lang="ts">
import { defineProps } from 'vue'

defineProps({
  device: {
    type: Object,
    required: true,
  },
})

function partitionSegments(device: any) {
  const map = [
    { key: 'system', class: 'storage-partition-system', label: 'System' },
    { key: 'documents', class: 'storage-partition-documents', label: 'Documents' },
    { key: 'media', class: 'storage-partition-media', label: 'Media' },
    { key: 'other', class: 'storage-partition-other', label: 'Other' },
    { key: 'free', class: 'storage-partition-free', label: 'Free' },
  ];
  const total = (device.used_bytes + device.avail_bytes) / 1_073_741_824;
  return map.map(seg => {
    let size = 0;
    if (seg.key === 'free') {
      size = device.avail_bytes / 1_073_741_824;
    } else if (device.categories && device.categories[seg.label]) {
      size = device.categories[seg.label] / 1_073_741_824 || 0;
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
/* Add styles for storage partition bar and legend here if needed */
</style>
