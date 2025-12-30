<template>
  <div class="storage-partition">
    <div class="storage-partition-info">
      <span class="storage-partition-label"
        >Capacity:
        {{ (device.total_bytes / 1_073_741_824).toFixed(1) }} GB</span
      >
      <span class="storage-partition-used"
        >{{ device.percent_used }}% used •
        {{ (device.used_bytes / 1_073_741_824).toFixed(1) }} GB</span
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
          v-if="device.avail_bytes > 0"
          class="storage-partition-segment storage-partition-free"
          :style="{ width: freeWidth }"
          :title="`Free: ${freeGB} GB`"
        />
      </template>
      <template v-else>
        <div
          v-if="device.percent_used > 0"
          class="storage-partition-segment storage-partition-used"
          :style="{ width: device.percent_used + '%' }"
          :title="`Used: ${(device.used_bytes / 1_073_741_824).toFixed(1)} GB`"
        />
        <div
          v-if="100 - device.percent_used > 0"
          class="storage-partition-segment storage-partition-free"
          :style="{ width: 100 - device.percent_used + '%' }"
          :title="`Free: ${(device.avail_bytes / 1_073_741_824).toFixed(1)} GB`"
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
      <div v-if="device.avail_bytes > 0" class="storage-partition-legend-item">
        <span class="storage-partition-dot storage-partition-free" />
        <span class="storage-partition-legend-label">Free {{ freeGB }} GB</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

defineProps({
  device: {
    type: Object,
    required: true,
  },
})

// Static test values for visual testing
const hasCategories = computed(() => {
  return true
})
type CategoryLabel =
  | 'System'
  | 'Documents'
  | 'Media'
  | 'Backups'
  | 'Other'
  | 'Free'

const testGB: Record<CategoryLabel, number> = {
  System: 40.81,
  Documents: 81.62,
  Media: 102.03,
  Backups: 0,
  Other: 183.65,
  Free: 19.9,
}
const totalGB =
  testGB.System + testGB.Documents + testGB.Media + testGB.Other + testGB.Free

const categorySegments = computed(() => {
  const map = [
    { key: 'system', class: 'storage-partition-system', label: 'System' },
    {
      key: 'documents',
      class: 'storage-partition-documents',
      label: 'Documents',
    },
    { key: 'media', class: 'storage-partition-media', label: 'Media' },
    { key: 'backups', class: 'storage-partition-backups', label: 'Backups' },
    { key: 'other', class: 'storage-partition-other', label: 'Other' },
  ]
  return map
    .map((seg) => {
      const gb = testGB[seg.label as CategoryLabel] || 0
      if (!gb || !totalGB)
        return {
          key: seg.key,
          class: seg.class,
          label: seg.label,
          width: '0%',
          title: '',
          gb: '0',
        }
      const percent = (gb / totalGB) * 100
      return {
        key: seg.key,
        class: seg.class,
        label: seg.label,
        width: percent.toFixed(2) + '%',
        title: `${seg.label}: ${gb.toFixed(1)} GB`,
        gb: gb.toFixed(1),
      }
    })
    .filter(Boolean)
})

const freeWidth = computed(() => {
  return ((testGB.Free / totalGB) * 100).toFixed(2) + '%'
})
const freeGB = computed(() => {
  return testGB.Free.toFixed(1)
})
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
