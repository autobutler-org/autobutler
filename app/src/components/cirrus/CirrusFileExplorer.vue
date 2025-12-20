<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { DeviceFileInfo } from '@/types/cirrus'
import { getFiles, getAvailableSpace, bytesToGB } from '@/services/cirrusService'
import CirrusBreadcrumb from './CirrusBreadcrumb.vue'
import CirrusListView from './CirrusListView.vue'
import CirrusGridView from './CirrusGridView.vue'

const route = useRoute()
const router = useRouter()

// State
const currentPath = ref('')
const view = ref<'list' | 'grid'>('list')
const showDeviceBadges = ref(false)

// Computed
const files = computed(() => getFiles(currentPath.value))
const availableSpace = computed(() => getAvailableSpace())
const availableSpaceFormatted = computed(() => `${bytesToGB(availableSpace.value).toFixed(2)}GB`)

// Watch route changes to update current path
watch(
  () => route.params.pathMatch,
  (newPath) => {
    if (Array.isArray(newPath)) {
      currentPath.value = newPath.join('/')
    } else {
      currentPath.value = newPath || ''
    }
  },
  { immediate: true },
)

// Methods
function navigateToPath(path: string) {
  currentPath.value = path
  router.push(`/cirrus${path ? '/' + path : ''}`)
}

function handleNavigateFolder(path: string) {
  navigateToPath(path)
}

function handleOpenFile(file: DeviceFileInfo) {
  // TODO: Implement file viewer
  console.log('Opening file:', file)
}

function switchView(newView: 'list' | 'grid') {
  view.value = newView
}

function toggleDeviceBadges(show: boolean) {
  showDeviceBadges.value = show
}
</script>

<template>
  <div id="file-explorer" class="file-explorer">
    <div class="file-explorer-header">
      <div>
        <h2 class="file-explorer-title">Cirrus</h2>
        <div class="file-explorer-space-info">Available Space: {{ availableSpaceFormatted }}</div>
      </div>
      <div style="display: flex; gap: 0.5rem; align-items: center">
        <!-- Navigation and download controls would go here -->
      </div>
    </div>

    <div id="file-explorer-selectable">
      <div class="file-explorer-controls">
        <div>
          <CirrusBreadcrumb :current-path="currentPath" @navigate="navigateToPath" />
        </div>
        <div style="display: flex; align-items: center; gap: 1rem">
          <label class="device-badge-toggle">
            <input
              type="checkbox"
              id="toggle-device-badges"
              :checked="showDeviceBadges"
              @change="toggleDeviceBadges(($event.target as HTMLInputElement).checked)"
            />
            <span>Show device names</span>
          </label>
          <div class="view-switcher">
            <button
              :class="['btn', 'btn--icon', view === 'list' ? 'btn--primary' : 'btn--secondary']"
              @click="switchView('list')"
              title="List View"
              type="button"
            >
              <!-- List view icon -->
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
                  d="M4 6h16M4 12h16M4 18h16"
                />
              </svg>
            </button>
            <button
              :class="['btn', 'btn--icon', view === 'grid' ? 'btn--primary' : 'btn--secondary']"
              @click="switchView('grid')"
              title="Grid View"
              type="button"
            >
              <!-- Grid view icon -->
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
                  d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zm10 0a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"
                />
              </svg>
            </button>
          </div>
        </div>
      </div>

      <div id="file-explorer-status"></div>

      <div id="file-explorer-view-content">
        <template v-if="files.length === 0">
          <span class="file-explorer-empty">No files found</span>
        </template>
        <template v-else-if="view === 'grid'">
          <CirrusGridView
            :files="files"
            :current-path="currentPath"
            @navigate-folder="handleNavigateFolder"
            @open-file="handleOpenFile"
          />
        </template>
        <template v-else>
          <CirrusListView
            :files="files"
            :current-path="currentPath"
            @navigate-folder="handleNavigateFolder"
            @open-file="handleOpenFile"
          />
        </template>
      </div>
    </div>
  </div>
</template>
