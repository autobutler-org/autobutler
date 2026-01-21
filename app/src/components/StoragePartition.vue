<template>
  <div class="storage-partition">
    <div class="storage-partition-info">
      <span class="storage-partition-label"
        >Capacity: {{ (device.totalBytes / GB).toFixed(1) }} GB</span
      >
      <span class="storage-partition-used"
        >{{ percentUsed }}% used •
        {{ (device.usedBytes / GB).toFixed(1) }} GB</span
      >
    </div>
    <div class="storage-partition-bar">
      <template v-if="hasCategories">
        <div
          v-for="cat in categorySegments"
          :key="cat.key"
          class="storage-partition-segment"
          :class="cat.class"
          :style="{ width: cat.width }"
          :title="cat.title"
        ></div>
        <div
          v-if="device.availableBytes > 0"
          class="storage-partition-segment storage-partition-free"
          :style="{ width: freeWidth }"
          :title="`Free: ${freeGB} GB`"
        />
      </template>
      <template v-else>
        <div
          v-if="percentUsed > 0"
          class="storage-partition-segment storage-partition-used"
          :style="{ width: percentUsed + '%' }"
          :title="`Used: ${(device.usedBytes / 1_073_741_824).toFixed(1)} GB`"
        />
        <div
          v-if="100 - percentUsed > 0"
          class="storage-partition-segment storage-partition-free"
          :style="{ width: 100 - percentUsed + '%' }"
          :title="`Free: ${(device.availableBytes / 1_073_741_824).toFixed(1)} GB`"
        />
      </template>
    </div>
    <div v-if="hasCategories" class="storage-partition-legend">
      <div
        v-for="cat in categorySegments"
        :key="cat.key"
        class="storage-partition-legend-item"
      >
        <span class="storage-partition-dot" :class="cat.class" />
        <span class="storage-partition-legend-label"
          >{{ cat.label }} {{ cat.gb }} GB</span
        >
      </div>
      <div
        v-if="device.availableBytes > 0"
        class="storage-partition-legend-item"
      >
        <span class="storage-partition-dot storage-partition-free" />
        <span class="storage-partition-legend-label">Free {{ freeGB }} GB</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Device } from '@/types/device';
import { computed } from 'vue';

const props = defineProps<{ device: Device }>();

const GB = 1024 * 1024 * 1024;

const percentUsed = computed(() => {
  const percent =
    props.device.totalBytes > 0
      ? (props.device.usedBytes / props.device.totalBytes) * 100
      : 0;
  return Number(percent.toFixed(2));
});

const hasCategories = computed(() => {
  return (
    props.device.categories && Object.keys(props.device.categories).length > 0
  );
});

const totalCategoryBytes = computed(() => {
  if (!props.device.categories) return 0;
  return Object.values(props.device.categories).reduce((sum, v) => sum + v, 0);
});

const categorySegments = computed(() => {
  if (!props.device.categories) return [];
  const totalBytes = totalCategoryBytes.value;
  if (!totalBytes) return [];
  // Use backend keys and values directly
  return Object.entries(props.device.categories)
    .map(([key, bytes]) => {
      if (!bytes) return null;
      const gb = bytes / GB;
      const percent = (bytes / totalBytes) * 100;
      // Use a class naming convention for color, fallback to 'other' if unknown
      const className = `storage-partition-${key}`;
      // Capitalize key for label
      const label = key.charAt(0).toUpperCase() + key.slice(1);
      return {
        key,
        class: className,
        label,
        width: percent.toFixed(2) + '%',
        title: `${label}: ${gb.toFixed(1)} GB`,
        gb: gb.toFixed(1),
      };
    })
    .filter((x) => !!x);
});

const freeWidth = computed(() => {
  const total = props.device.totalBytes || 0;
  const free = props.device.availableBytes || 0;
  if (!total) return '0%';
  return ((free / total) * 100).toFixed(2) + '%';
});
const freeGB = computed(() => {
  return (props.device.availableBytes / GB).toFixed(1);
});
</script>

<style lang="scss" scoped>
.storage-partition {
  width: 100%;
}

.storage-partition-bar {
  display: flex;
  height: 24px;
  background: $theme-palette-bg-primary;
  border-radius: $border-radius;
  overflow: hidden;
  margin-bottom: $spacing-md;
  border: 1px solid $theme-palette-border;

  @media (prefers-color-scheme: light) {
    background: $theme-palette-bg-inverse;
    border: 1px solid $theme-palette-border-strong;
  }

  @media (max-width: 768px) {
    height: 20px;
  }
}

.storage-partition-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.storage-partition-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: $spacing-sm;
  font-size: $theme-font-size-sm;
}

.storage-partition-label {
  color: $theme-palette-text-secondary;
  font-weight: 500;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-inverse;
  }
}

.storage-partition-used {
  color: $theme-palette-text-muted;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-secondary;
  }
}

.storage-partition-segment {
  transition: all 0.3s ease;
  cursor: pointer;
  position: relative;

  &:hover {
    opacity: 0.8;
    filter: brightness(1.1);
  }
}

/* Category Colors - matching existing device card colors */
.storage-partition-backups {
  background: $color-yellow-500; /* Orange */
}

.storage-partition-documents {
  background: $color-purple-400; /* Purple */
}

.storage-partition-free {
  background: $theme-palette-border; /* Subtle border/dark gray */
}

.storage-partition-media {
  background: $color-red-400; /* Pink/Red */
}

.storage-partition-other {
  background: $color-yellow-400; /* Yellow */
}

.storage-partition-system {
  background: $color-blue-500; /* Blue */
}

.storage-partition-legend {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-md;

  @media (max-width: 768px) {
    gap: $spacing-sm;
  }
}

.storage-partition-legend-item {
  display: flex;
  align-items: center;
  gap: $spacing-xs;
}

.storage-partition-legend-label {
  font-size: 0.8125rem;
  color: $theme-palette-text-muted;
  white-space: nowrap;

  @media (prefers-color-scheme: light) {
    color: $theme-palette-text-secondary;
  }

  @media (max-width: 768px) {
    font-size: 0.75rem;
  }
}
</style>
