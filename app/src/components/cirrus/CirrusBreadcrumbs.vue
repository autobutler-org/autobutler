<template>
  <nav class="file-explorer-breadcrumbs" :data-path="currentPath">
    <span
      v-for="(segment, index) in segments"
      :key="index"
      class="file-explorer-breadcrumb"
      :class="{
        'breadcrumb--drag-hover':
          hoveredBreadcrumb === segment.path && index !== segments.length - 1,
      }"
      @dragenter="handleBreadcrumbDragEnter($event, segment.path)"
      @dragover="handleBreadcrumbDragOver($event, segment.path)"
      @dragleave="handleBreadcrumbDragLeave($event, segment.path)"
      @drop="handleBreadcrumbDrop($event, segment.path)"
    >
      <a href="#" @click.prevent="navigateTo(segment.path)">
        {{ segment.name }}
      </a>
      <span>/</span>
    </span>
    <CirrusAddFolder
      :current-path="currentPath"
      @folder-created="handleFolderCreated"
    />
  </nav>
</template>

<script lang="ts" setup>
import { normalizePath } from '@/util/filepath';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import CirrusAddFolder from './CirrusAddFolder.vue';

const props = defineProps<{
  currentPath: string;
}>();

const emit = defineEmits<{
  navigate: [path: string];
  'folder-created': [folderName: string];
  'file-move': [
    oldPath: string,
    newPath: string,
    oldDeviceSerial?: string,
    newDeviceSerial?: string,
  ];
}>();
const hoveredBreadcrumb = ref<string | null>(null);

const handleBreadcrumbDragEnter = (event: DragEvent, path: string) => {
  event.preventDefault();
  hoveredBreadcrumb.value = path;
};

const handleBreadcrumbDragOver = (event: DragEvent, path: string) => {
  event.preventDefault();
  hoveredBreadcrumb.value = path;
};

const handleBreadcrumbDragLeave = (event: DragEvent, _: string) => {
  event.preventDefault();
  hoveredBreadcrumb.value = null;
};

const handleBreadcrumbDrop = (dropEvent: DragEvent, path: string) => {
  dropEvent.preventDefault();
  dropEvent.stopPropagation();

  hoveredBreadcrumb.value = null;
  // Do not emit if dropped on the last breadcrumb (current location)
  path = normalizePath(path);
  const targetPath = normalizePath(
    (dropEvent.target as HTMLAnchorElement).pathname,
  ).replace(/^\/cirrus\//, '');
  if (path === targetPath) {
    return;
  }
  emit('file-move', path, targetPath);
};

const router = useRouter();

// Parse the path into breadcrumb segments
const segments = computed(() => {
  const parts = props.currentPath.split('/').filter((p) => {
    return p.length > 0;
  });
  const result = [{ name: 'cirrus', path: '' }];

  let accumulatedPath = '';
  for (const part of parts) {
    accumulatedPath = accumulatedPath ? `${accumulatedPath}/${part}` : part;
    result.push({ name: part, path: accumulatedPath });
  }

  return result;
});

const handleFolderCreated = (folderName: string) => {
  emit('folder-created', folderName);
};

const navigateTo = (path: string) => {
  emit('navigate', path);
  router.push(`/cirrus${path ? '/' + path : ''}`);
};
</script>

<style lang="scss" scoped>
.file-explorer-breadcrumbs {
  display: flex;
  align-items: center;
  gap: $spacing-sm;
  font-size: $theme-font-size-sm;
  color: $theme-palette-text-muted;
  margin-bottom: $spacing-sm;
}

.file-explorer-breadcrumb {
  color: $theme-palette-text-primary;

  a {
    color: inherit;
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }
}

.breadcrumb--drag-hover {
  background: $theme-palette-bg-secondary;
  color: $theme-palette-accent;
  border-radius: 4px;
  box-shadow: 0 0 0 2px $theme-palette-accent;
}
</style>
