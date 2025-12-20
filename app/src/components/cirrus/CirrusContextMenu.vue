<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import type { CirrusFileNode } from '@/types/cirrus'

const props = defineProps<{
  modelValue: boolean
  file: CirrusFileNode | null
  currentPath: string
  x: number
  y: number
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  download: [file: CirrusFileNode]
  rename: [file: CirrusFileNode]
  details: [file: CirrusFileNode]
  delete: [file: CirrusFileNode]
}>()

const menuRef = ref<HTMLElement | null>(null)

function closeMenu() {
  emit('update:modelValue', false)
}

function handleClickOutside(event: MouseEvent) {
  if (menuRef.value && !menuRef.value.contains(event.target as Node)) {
    closeMenu()
  }
}

function addListeners() {
  // Use setTimeout to avoid the current event from triggering close
  setTimeout(() => {
    document.addEventListener('click', handleClickOutside)
    document.addEventListener('contextmenu', handleClickOutside)
  }, 0)
}

function removeListeners() {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('contextmenu', handleClickOutside)
}

function handleDownload() {
  if (props.file) {
    emit('download', props.file)
  }
  closeMenu()
}

function handleRename() {
  if (props.file) {
    emit('rename', props.file)
  }
  closeMenu()
}

function handleDetails() {
  if (props.file) {
    emit('details', props.file)
  }
  closeMenu()
}

function handleDelete() {
  if (props.file) {
    emit('delete', props.file)
  }
  closeMenu()
}

// Adjust position to keep menu on screen
async function adjustPosition() {
  await nextTick()
  if (!menuRef.value) return

  const rect = menuRef.value.getBoundingClientRect()
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight

  // Adjust X if menu goes off right edge
  if (rect.right > viewportWidth) {
    menuRef.value.style.left = `${viewportWidth - rect.width - 10}px`
  }

  // Adjust Y if menu goes off bottom edge
  if (rect.bottom > viewportHeight) {
    menuRef.value.style.top = `${viewportHeight - rect.height - 10}px`
  }
}

onMounted(() => {
  // Listeners are added/removed dynamically based on menu open state
})

onUnmounted(() => {
  removeListeners()
})

// Watch for menu opening to adjust position and add/remove listeners
watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen) {
      addListeners()
      adjustPosition()
    } else {
      removeListeners()
    }
  }
)

// Also watch x and y changes in case they update while open
watch(
  () => [props.x, props.y],
  () => {
    if (props.modelValue) {
      adjustPosition()
    }
  }
)
</script>

<template>
  <Teleport to="body">
    <div
      v-if="modelValue && file"
      ref="menuRef"
      class="context-menu"
      :style="{ left: `${x}px`, top: `${y}px` }"
      @contextmenu.prevent
    >
      <ul class="context-menu-list">
        <li>
          <button type="button" class="context-menu-item" @click="handleDownload">
            Download
          </button>
        </li>
        <li>
          <button type="button" class="context-menu-item" @click="handleRename">
            Move/Rename
          </button>
        </li>
        <li>
          <button type="button" class="context-menu-item" @click="handleDetails">
            File Details
          </button>
        </li>
        <li>
          <button type="button" class="context-menu-item context-menu-item--danger" @click="handleDelete">
            Delete
          </button>
        </li>
      </ul>
    </div>
  </Teleport>
</template>

<style lang="scss">
/* Not scoped because Teleport renders outside component tree */
.context-menu {
  position: fixed;
  background-color: white;
  border: 1px solid var(--color-gray-200);
  border-radius: var(--border-radius);
  box-shadow: var(--shadow-lg);
  z-index: 10000;
  min-width: 140px;

  @media (prefers-color-scheme: dark) {
    background-color: var(--color-gray-800);
    border-color: var(--color-gray-700);
  }
}

.context-menu-list {
  list-style: none;
  margin: 0;
  padding: var(--spacing-xs) 0;
}

.context-menu-item {
  width: 100%;
  text-align: left;
  padding: var(--spacing-sm) var(--spacing-lg);
  border: none;
  background: transparent;
  font-size: var(--font-size-sm);
  cursor: pointer;
  display: block;
  text-decoration: none;
  color: var(--color-gray-900);

  &:hover {
    background-color: var(--color-gray-100);
  }

  @media (prefers-color-scheme: dark) {
    color: var(--color-gray-100);

    &:hover {
      background-color: var(--color-gray-700);
    }
  }
}

.context-menu-item--danger {
  color: white;
  background-color: var(--color-red-800);

  &:hover {
    background-color: var(--color-red-600);
  }
}
</style>
