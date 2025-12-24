<template>
  <nav class="file-explorer-breadcrumbs" :data-path="currentPath">
    <span v-for="(segment, index) in segments" :key="index" class="file-explorer-breadcrumb">
      <a href="#" @click.prevent="navigateTo(segment.path)">
        {{ segment.name }}
      </a>
      <span>/</span>
    </span>
    <CirrusAddFolder :current-path="currentPath"
    @folder-created="handleFolderCreated"></CirrusAddFolder>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import CirrusAddFolder from '@/components/cirrus/CirrusAddFolder.vue';

const props = defineProps<{
  currentPath: string
}>()

const emit = defineEmits<{
  navigate: [path: string]
  'folder-created': [folderName: string]
}>()

const router = useRouter()

// Parse the path into breadcrumb segments
const segments = computed(() => {
  const parts = props.currentPath.split('/').filter((p) => {
    return p.length > 0
  })
  const result = [{ name: 'cirrus', path: '' }]

  let accumulatedPath = ''
  for (const part of parts) {
    accumulatedPath = accumulatedPath ? `${accumulatedPath}/${part}` : part
    result.push({ name: part, path: accumulatedPath })
  }

  return result
})

const handleFolderCreated = (folderName: string) => {
  emit('folder-created', folderName)
}

const navigateTo = (path: string) => {
  emit('navigate', path)
  router.push(`/cirrus${path ? '/' + path : ''}`)
}
</script>

<style lang="scss" scoped>
.file-explorer-breadcrumbs {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  font-size: $font-size-sm;
  color: $color-gray-500;
  margin-bottom: $spacing-sm;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-400;
  }
}

.file-explorer-breadcrumb {
  color: $color-gray-700;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-300;
  }

  a {
    color: inherit;
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }
}
</style>
