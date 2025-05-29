<template>
  <nav class="table-of-contents" v-if="headings.length > 0">
    <ul class="toc-list">
      <li 
        v-for="heading in headings" 
        :key="heading.id"
        :class="[
          'toc-item',
          `toc-level-${heading.level}`,
          { 'toc-active': activeId === heading.id }
        ]"
      >
        <a 
          :href="`#${heading.id}`"
          class="toc-link"
          @click="handleLinkClick(heading.id, $event)"
        >
          {{ heading.text }}
        </a>
      </li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
import { useScrollSpy } from '~/composables/useScrollSpy'

const { activeId, headings, scrollToHeading, refresh } = useScrollSpy()

const handleLinkClick = (id: string, event: Event) => {
  event.preventDefault()
  scrollToHeading(id)
}

// Refresh when content changes (useful for dynamic content)
defineExpose({
  refresh
})
</script>

<style scoped>
.table-of-contents {
  font-size: 0.875rem;
}

.toc-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.toc-item {
  margin: 0;
  position: relative;
}

.toc-link {
  display: block;
  color: rgba(255, 255, 255, 0.6);
  text-decoration: none;
  padding: 0.375rem 0;
  border-radius: 0.25rem;
  transition: all 0.2s ease;
  position: relative;
  line-height: 1.4;
}

.toc-link:hover {
  color: rgba(255, 255, 255, 0.9);
  background: rgba(255, 255, 255, 0.05);
  padding-left: 0.5rem;
}

/* Active state */
.toc-active .toc-link {
  color: rgba(0, 255, 170, 0.9);
  background: rgba(0, 255, 170, 0.1);
  padding-left: 0.75rem;
  font-weight: 500;
}

.toc-active .toc-link::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 60%;
  background: rgba(0, 255, 170, 0.8);
  border-radius: 2px;
}

/* Heading level indentation */
.toc-level-1 .toc-link {
  padding-left: 0;
  font-weight: 600;
  font-size: 0.9rem;
}

.toc-level-2 .toc-link {
  padding-left: 1rem;
}

.toc-level-3 .toc-link {
  padding-left: 2rem;
  font-size: 0.8rem;
}

.toc-level-4 .toc-link {
  padding-left: 2.5rem;
  font-size: 0.8rem;
}

.toc-level-5 .toc-link {
  padding-left: 3rem;
  font-size: 0.75rem;
}

.toc-level-6 .toc-link {
  padding-left: 3.5rem;
  font-size: 0.75rem;
}

/* Adjust active state padding for indented items */
.toc-level-2.toc-active .toc-link {
  padding-left: 1.25rem;
}

.toc-level-3.toc-active .toc-link {
  padding-left: 2.25rem;
}

.toc-level-4.toc-active .toc-link {
  padding-left: 2.75rem;
}

.toc-level-5.toc-active .toc-link {
  padding-left: 3.25rem;
}

.toc-level-6.toc-active .toc-link {
  padding-left: 3.75rem;
}

/* Hover state adjustments for indented items */
.toc-level-2 .toc-link:hover {
  padding-left: 1.5rem;
}

.toc-level-3 .toc-link:hover {
  padding-left: 2.5rem;
}

.toc-level-4 .toc-link:hover {
  padding-left: 3rem;
}

.toc-level-5 .toc-link:hover {
  padding-left: 3.5rem;
}

.toc-level-6 .toc-link:hover {
  padding-left: 4rem;
}

/* Smooth transitions for all states */
.toc-link {
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

/* Mobile adjustments */
@media (max-width: 768px) {
  .table-of-contents {
    font-size: 0.8rem;
  }
  
  .toc-level-3 .toc-link,
  .toc-level-4 .toc-link,
  .toc-level-5 .toc-link,
  .toc-level-6 .toc-link {
    padding-left: 1.5rem;
    font-size: 0.8rem;
  }
  
  .toc-level-3.toc-active .toc-link,
  .toc-level-4.toc-active .toc-link,
  .toc-level-5.toc-active .toc-link,
  .toc-level-6.toc-active .toc-link {
    padding-left: 1.75rem;
  }
  
  .toc-level-3 .toc-link:hover,
  .toc-level-4 .toc-link:hover,
  .toc-level-5 .toc-link:hover,
  .toc-level-6 .toc-link:hover {
    padding-left: 2rem;
  }
}
</style> 