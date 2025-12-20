<template>
  <div class="storage-partition-component">
    <div class="storage-partition-info">
      <span class="storage-partition-label">
        Capacity: {{ (device.total_bytes / 1_073_741_824).toFixed(1) }} GB
      </span>
      <span class="storage-partition-used">
        {{ device.percent_used }}% used • {{ (device.used_bytes / 1_073_741_824).toFixed(1) }} GB
      </span>
    </div>
    <div class="storage-partition-bar">
      <template v-if="Object.keys(device.categories || {}).length > 0">
        <template v-for="[category, bytes] in categoryEntries" :key="category">
          <div
            class="storage-partition-segment"
            :class="segmentClass(category)"
            :style="{ width: segmentWidth(category, bytes) }"
            :title="segmentTitle(category, bytes)"
          ></div>
        </template>
        <div
          v-if="device.avail_bytes > 0"
          class="storage-partition-segment storage-partition-free"
          :style="{ width: freeWidth }"
          :title="`Free: ${(device.avail_bytes / 1_073_741_824).toFixed(1)} GB`"
        ></div>
      </template>
      <template v-else>
        <div
          v-if="device.percent_used > 0"
          class="storage-partition-segment storage-partition-used"
          :style="{ width: device.percent_used + '%' }"
          :title="`Used: ${(device.used_bytes / 1_073_741_824).toFixed(1)} GB`"
        ></div>
        <div
          v-if="100 - device.percent_used > 0"
          class="storage-partition-segment storage-partition-free"
          :style="{ width: (100 - device.percent_used) + '%' }"
          :title="`Free: ${(device.avail_bytes / 1_073_741_824).toFixed(1)} GB`"
        ></div>
      </template>
    </div>
    <div v-if="Object.keys(device.categories || {}).length > 0" class="storage-partition-legend">
      <template v-for="[category, bytes] in categoryEntries" :key="category">
        <div class="storage-partition-legend-item">
          <span class="storage-partition-dot" :class="segmentClass(category)"></span>
          <span class="storage-partition-legend-label">
            {{ legendLabel(category, bytes) }}
          </span>
        </div>
      </template>
      <div v-if="device.avail_bytes > 0" class="storage-partition-legend-item">
        <span class="storage-partition-dot storage-partition-free"></span>
        <span class="storage-partition-legend-label">
          Free {{ (device.avail_bytes / 1_073_741_824).toFixed(1) }} GB
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">

import { computed } from 'vue';


const props = defineProps({
  device: {
    type: Object,
    required: true,
  },
});

const categoryEntries = computed<[string, number][]>(() => {
  return Object.entries(props.device.categories || {});
});

const totalForCalc = computed(() => props.device.used_bytes + props.device.avail_bytes);

function segmentClass(category: string) {
  switch (category) {
    case 'system': return 'storage-partition-system';
    case 'documents': return 'storage-partition-documents';
    case 'media': return 'storage-partition-media';
    case 'backups': return 'storage-partition-backups';
    case 'other': return 'storage-partition-other';
    default: return '';
  }
}

function segmentWidth(category: string, bytes: number) {
  if (!totalForCalc.value) return '0%';
  return ((bytes / totalForCalc.value) * 100).toFixed(2) + '%';
}

function segmentTitle(category: string, bytes: number) {
  const gb = (bytes / 1_073_741_824).toFixed(2);
  switch (category) {
    case 'system': return `System: ${gb} GB`;
    case 'documents': return `Documents: ${gb} GB`;
    case 'media': return `Media: ${gb} GB`;
    case 'backups': return `Backups: ${gb} GB`;
    case 'other': return `Other: ${gb} GB`;
    default: return `${category}: ${gb} GB`;
  }
}

const freeWidth = computed(() => {
  if (!totalForCalc.value) return '0%';
  return ((props.device.avail_bytes / totalForCalc.value) * 100).toFixed(2) + '%';
});

function legendLabel(category: string, bytes: number) {
  const gb = (bytes / 1_073_741_824).toFixed(1);
  switch (category) {
    case 'system': return `System ${gb} GB`;
    case 'documents': return `Documents ${gb} GB`;
    case 'media': return `Media ${gb} GB`;
    case 'backups': return `Backups ${gb} GB`;
    case 'other': return `Other ${gb} GB`;
    default: return `${category} ${gb} GB`;
  }
}
</script>

<style scoped>
</style>
