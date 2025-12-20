<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'

const props = defineProps<{
  currentPath: string
}>()

const emit = defineEmits<{
  navigate: [path: string]
  'add-folder': []
}>()

const router = useRouter()

// Parse the path into breadcrumb segments
const segments = computed(() => {
  const parts = props.currentPath.split('/').filter((p) => p.length > 0)
  const result = [{ name: 'cirrus', path: '' }]

  let accumulatedPath = ''
  for (const part of parts) {
    accumulatedPath = accumulatedPath ? `${accumulatedPath}/${part}` : part
    result.push({ name: part, path: accumulatedPath })
  }

  return result
})

function navigateTo(path: string) {
  emit('navigate', path)
  router.push(`/cirrus${path ? '/' + path : ''}`)
}
</script>

<template>
  <nav class="file-explorer-breadcrumbs" :data-path="currentPath">
    <span v-for="(segment, index) in segments" :key="index" class="file-explorer-breadcrumb">
      <a href="#" @click.prevent="navigateTo(segment.path)">
        {{ segment.name }}
      </a>
      <span>/</span>
    </span>
    <div class="file-explorer-folder-controls">
      <button
        id="add-folder-btn"
        class="file-explorer-add-folder btn btn--icon"
        title="Add Folder"
        type="button"
        @click="$emit('add-folder')"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="icon icon--base"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 4v16m8-8H4"
          />
        </svg>
      </button>
    </div>
  </nav>
</template>
