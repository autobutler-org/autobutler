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
          <button
            type="button"
            class="context-menu-item"
            @click="handleDownload"
          >
            Download
          </button>
        </li>
        <li>
          <button type="button" class="context-menu-item" @click="handleRename">
            Move/Rename
          </button>
        </li>
        <li>
          <button
            type="button"
            class="context-menu-item"
            @click="handleShowDetails"
          >
            File Details
          </button>
        </li>
        <li>
          <button
            type="button"
            class="context-menu-item context-menu-item--danger"
            @click="handleDelete"
          >
            Delete
          </button>
        </li>
      </ul>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import type { CirrusFileNode } from '@/types/cirrus'
import { nextTick, onUnmounted, ref, watch } from 'vue'

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

const addListeners = () => {
  // Use setTimeout to avoid the current event from triggering close
  setTimeout(() => {
    document.addEventListener('click', handleClickOutside)
    document.addEventListener('contextmenu', handleClickOutside)
  }, 0)
}

// Adjust position to keep menu on screen
const adjustPosition = async () => {
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

const closeMenu = () => {
  emit('update:modelValue', false)
}

const handleClickOutside = (event: MouseEvent) => {
  if (menuRef.value && !menuRef.value.contains(event.target as Node)) {
    closeMenu()
  }
}

const handleDelete = () => {
  if (props.file) {
    emit('delete', props.file)
  }
  closeMenu()
}

const handleDownload = () => {
  if (props.file) {
    emit('download', props.file)
  }
  closeMenu()
}

const handleRename = () => {
  if (props.file) {
    emit('rename', props.file)
  }
  closeMenu()
}

const handleShowDetails = () => {
  if (props.file) {
    emit('details', props.file)
  }
  closeMenu()
}

const removeListeners = () => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('contextmenu', handleClickOutside)
}

onUnmounted(() => {
  removeListeners()
})

// Watch for menu opening to adjust position and add/remove listeners
watch(
  () => {
    return props.modelValue
  },
  (isOpen) => {
    if (isOpen) {
      addListeners()
      adjustPosition()
    } else {
      removeListeners()
    }
  },
)

// Also watch x and y changes in case they update while open
watch(
  () => {
    return [props.x, props.y]
  },
  () => {
    if (props.modelValue) {
      adjustPosition()
    }
  },
)
</script>

<style lang="scss">
/* Not scoped because Teleport renders outside component tree */
.context-menu {
  position: fixed;
  background-color: $color-gray-50;
  border: 1px solid $color-gray-200;
  border-radius: $border-radius;
  box-shadow: $shadow-lg;
  z-index: 10000;
  min-width: 140px;

  @media (prefers-color-scheme: dark) {
    background-color: $color-gray-800;
    border-color: $color-gray-700;
  }
}

.context-menu-list {
  list-style: none;
  margin: 0;
  padding: $spacing-xs 0;
}

.context-menu-item {
  width: 100%;
  text-align: left;
  padding: $spacing-sm $spacing-lg;
  border: none;
  background: transparent;
  font-size: $theme-font-size-sm;
  cursor: pointer;
  display: block;
  text-decoration: none;
  color: $color-gray-900;

  &:hover {
    background-color: $color-gray-100;
  }

  @media (prefers-color-scheme: dark) {
    color: $color-gray-100;

    &:hover {
      background-color: $color-gray-700;
    }
  }
}

.context-menu-item--danger {
  color: $color-red-400;

  @media (prefers-color-scheme: light) {
    color: $color-red-600;
  }

  &:hover {
    background-color: rgba($color-red-800, 0.7);

    @media (prefers-color-scheme: light) {
      background-color: rgba($color-red-200, 0.7);
    }
  }
}
</style>
