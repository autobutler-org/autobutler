<template>
  <nav class="library-sidebar">
    <div v-for="section in sections" :key="section.title" class="library-sidebar-section">
      <h2 class="library-sidebar-title">{{ section.title }}</h2>
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

<script setup lang="ts">
interface SidebarItem {
  label: string
  count?: number
  active?: boolean
}
interface SidebarSection {
  title: string
  items: SidebarItem[]
}

defineProps<{ sections: SidebarSection[] }>()
</script>

<style lang="scss" scoped>
/* Match PhotosSidebar styles */
.library-sidebar {
  width: 100%;
  height: 100%;
  background: var(--color-gray-900);
  color: white;
  padding: var(--spacing-xl);
  overflow-y: auto;
  flex-shrink: 0;
  border-right: 1px solid var(--color-gray-800);
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}
.library-sidebar-section {
  margin-bottom: var(--spacing-2xl);
}
.library-sidebar-title {
  font-size: var(--font-size-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-gray-400);
  margin-bottom: var(--spacing-md);
}
.library-sidebar-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}
.library-sidebar-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-md);
  border-radius: var(--border-radius);
  color: var(--color-gray-300);
  text-decoration: none;
  transition: all 0.2s ease;
  font-size: var(--font-size-sm);
  cursor: pointer;
}
.library-sidebar-item:hover {
  background: var(--color-gray-800);
  color: white;
}
.library-sidebar-item.active {
  background: var(--color-primary-600);
  color: white;
}
.library-sidebar-count {
  margin-left: auto;
  font-size: var(--font-size-xs);
  color: var(--color-gray-500);
}
.library-sidebar-item.active .library-sidebar-count {
  color: var(--color-primary-100);
}
</style>
