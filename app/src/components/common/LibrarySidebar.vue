<template>
  <nav class="library-sidebar">
    <div
      v-for="section in sections"
      :key="section.title"
      class="library-sidebar-section"
    >
      <h2 class="library-sidebar-title">
        {{ section.title }}
      </h2>
      <ul class="library-sidebar-list">
        <li
          v-for="item in section.items"
          :key="item.label"
          :class="['library-sidebar-item', { active: item.active }]"
        >
          <span>{{ item.label }}</span>
          <span v-if="item.count !== undefined" class="library-sidebar-count">{{
            item.count
          }}</span>
        </li>
      </ul>
    </div>
  </nav>
</template>

<script lang="ts" setup>
export interface SidebarItem {
  label: string
  count?: number
  active?: boolean
}
export interface SidebarSection {
  title: string
  items: SidebarItem[]
}

defineProps<{ sections: SidebarSection[] }>()
</script>

<style lang="scss" scoped>
.library-sidebar {
  width: 100%;
  height: 100%;
  background: $color-gray-900;
  color: white;
  padding: $spacing-xl;
  overflow-y: auto;
  flex-shrink: 0;
  border-right: 1px solid $color-gray-800;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

.library-sidebar-count {
  margin-left: auto;
  font-size: $theme-font-size-xs;
  color: $color-gray-100;
}

.library-sidebar-item {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: $spacing-sm $spacing-md;
  border-radius: $border-radius;
  color: $color-gray-300;
  text-decoration: none;
  transition: all 0.2s ease;
  font-size: $theme-font-size-sm;
  cursor: pointer;

  & .active {
    color: $color-primary-100;
  }

  &:hover {
    background: $color-gray-800;
    color: white;
  }

  &.active {
    background: $color-primary-600;
    color: white;
  }
}

.library-sidebar-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: $spacing-xs;
}

.library-sidebar-section {
  margin-bottom: $spacing-2xl;
}

.library-sidebar-title {
  font-size: $theme-font-size-xs;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: $color-gray-400;
  margin-bottom: $spacing-md;
}
</style>
