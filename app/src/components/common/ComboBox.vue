<template>
  <div class="combobox" @keydown.esc="close">
    <div class="control" @click.self="toggle" :class="{ open: isOpen }">
      <div class="tags" v-if="selected.length">
        <span
          class="tag"
          v-for="s in selected"
          :key="s.value"
          :class="{ active: activeChipValue === s.value }"
          @click.stop="deselect(s.value)"
        >
          {{ s.label }}
        </span>
      </div>
      <input
        v-model="query"
        :placeholder="selected.length ? '' : placeholder"
        @focus="open"
        @input="onInput"
        @keydown="onInputKeydown"
        class="input"
        type="text"
        aria-haspopup="listbox"
        :aria-expanded="isOpen"
      />
      <button class="chev" @click.stop="toggle" aria-hidden>▾</button>
    </div>

    <ul v-if="isOpen" class="menu" role="listbox">
      <li
        v-for="item in filteredItems"
        :key="item.value"
        :class="{ selected: isSelected(item.value) }"
        @click="toggleSelect(item.value)"
        role="option"
        :aria-selected="isSelected(item.value)"
      >
        <input type="checkbox" :checked="isSelected(item.value)" readonly />
        <span class="label">{{ item.label }}</span>
      </li>
      <li v-if="!filteredItems.length" class="empty">No results</li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
import { computed, getCurrentInstance, onMounted, ref, watch } from 'vue';

interface Item {
  value: string | number;
  label: string;
}

const props = defineProps<{
  items: Item[];
  modelValue?: Array<string | number>;
  placeholder?: string;
  filterBy?: (item: Item, q: string) => boolean;
}>();

const emit = defineEmits(['update:modelValue']);

const isOpen = ref(false);
const query = ref('');
const internal = ref<Item[]>(props.items || []);
const activeChipIndex = ref(-1);

const selected = computed(() => {
  const vals = props.modelValue || [];
  return internal.value.filter((i) => vals.includes(i.value));
});

const activeChipValue = computed(() => {
  if (activeChipIndex.value < 0) return null;
  return selected.value[activeChipIndex.value]?.value ?? null;
});

watch(
  () => props.items,
  (v) => (internal.value = v || []),
);

const placeholder = props.placeholder ?? 'Select...';

const ignoreDocumentClick = ref(false);

const open = () => {
  isOpen.value = true;
  // ignore the next document click that may be caused by the mousedown/click that opened the input
  ignoreDocumentClick.value = true;
  setTimeout(() => (ignoreDocumentClick.value = false), 0);
};
const close = () => {
  isOpen.value = false;
};
const toggle = () => {
  isOpen.value = !isOpen.value;
};

const isSelected = (val: string | number) =>
  (props.modelValue || []).includes(val);

const select = (val: string | number) => {
  const prev = Array.from(props.modelValue || []);
  if (!prev.includes(val)) {
    prev.push(val);
    emit('update:modelValue', prev);
  }
};

const deselect = (val: string | number) => {
  const prev = Array.from(props.modelValue || []);
  const idx = prev.indexOf(val);
  if (idx !== -1) {
    prev.splice(idx, 1);
    emit('update:modelValue', prev);
  }
};

const toggleSelect = (val: string | number) => {
  if (isSelected(val)) deselect(val);
  else select(val);
};

const onInput = () => {
  activeChipIndex.value = -1;
  if (!isOpen.value) isOpen.value = true;
};

const onInputKeydown = (e: KeyboardEvent) => {
  if (!selected.value.length) return;

  if (e.key === 'ArrowLeft' && !query.value.length) {
    e.preventDefault();
    if (activeChipIndex.value === -1) {
      activeChipIndex.value = selected.value.length - 1;
      return;
    }
    activeChipIndex.value = Math.max(0, activeChipIndex.value - 1);
    return;
  }

  if (e.key === 'ArrowRight' && activeChipIndex.value !== -1) {
    e.preventDefault();
    if (activeChipIndex.value >= selected.value.length - 1) {
      activeChipIndex.value = -1;
      return;
    }
    activeChipIndex.value += 1;
    return;
  }

  if (e.key !== 'Backspace' || query.value.length) return;

  e.preventDefault();
  if (activeChipIndex.value === -1) {
    const last = selected.value[selected.value.length - 1];
    if (!last) return;
    deselect(last.value);
    return;
  }

  const indexToRemove = activeChipIndex.value;
  const chip = selected.value[indexToRemove];
  if (!chip) {
    activeChipIndex.value = -1;
    return;
  }

  deselect(chip.value);
  activeChipIndex.value = Math.max(-1, indexToRemove - 1);
};

watch(
  () => props.modelValue,
  () => {
    if (!selected.value.length) {
      activeChipIndex.value = -1;
      return;
    }

    if (activeChipIndex.value > selected.value.length - 1) {
      activeChipIndex.value = selected.value.length - 1;
    }
  },
);

const filteredItems = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return internal.value;
  const fn =
    props.filterBy ||
    ((item: Item, s: string) => item.label.toLowerCase().includes(s));
  return internal.value.filter((i) => fn(i, q));
});

// click outside directive polyfill for simple cases
onMounted(() => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const root = (getCurrentInstance() as any)?.vnode?.el as
    | HTMLElement
    | undefined;
  if (!root) return;
  const handler = (e: MouseEvent) => {
    if (ignoreDocumentClick.value) return;
    if (!root.contains(e.target as Node)) close();
  };
  document.addEventListener('click', handler);
});
</script>

<style scoped lang="scss">
.combobox {
  background: $theme-palette-component-primary;
  border-radius: $border-radius-md;
  border: 0.09rem solid $theme-palette-border-dark;
  color: $theme-palette-text-inverse;
  flex: 0 1 auto;
  font-family: inherit;
  font-size: $theme-font-size-base;
  margin-right: 0.5rem;
  max-width: 100%;
  min-width: 18rem;
  padding: 0.4rem 0.7rem;
  position: relative;
  width: 100%;
  .control {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    border: 0.0625rem solid $theme-palette-border-dark;
    padding: 0.375rem 0.5rem;
    border-radius: 0.375rem;
    background: $theme-palette-component-primary;
    cursor: text;
    &.open {
      box-shadow: 0 0.25rem 0.75rem rgba(0, 0, 0, 0.08);
    }
    .tags {
      display: flex;
      gap: 0.375rem;
      flex-wrap: wrap;
      flex: 0 1 auto;
      align-items: center;
      /* allow input to shrink properly */
      .tag {
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        background: $theme-palette-accent-hover;
        color: $theme-palette-text-inverse;
        padding: 0.125rem 0.375rem;
        border-radius: 999px;
        font-size: 0.75rem;
        white-space: nowrap;

        &:hover {
          background: $theme-palette-accent;
          color: $theme-palette-text-inverse;
        }

        &.active {
          background: $theme-palette-accent;
          color: $theme-palette-text-inverse;
        }
      }
    }
    .input {
      flex: 1 1 auto;
      border: 0;
      outline: none;
      min-width: 2.25rem;
      min-width: 0; /* ensure it can shrink */
      background: transparent;
      color: inherit;
      &::placeholder {
        color: $theme-palette-text-muted;
        opacity: 1;
      }
    }
    .chev {
      background: transparent;
      border: none;
      cursor: pointer;
      padding: 0.125rem 0.375rem;
    }
  }

  .menu {
    position: absolute;
    z-index: 40;
    left: 0;
    right: 0;
    margin-top: 0.375rem; /* 6px */
    border: 0.0625rem solid $theme-palette-border-strong; /* 1px */
    background: $theme-palette-component-primary;
    border-radius: 0.5rem;
    max-height: 13.75rem;
    overflow: auto;
    list-style: none;
    padding: 0.375rem 0.25rem;

    li {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.5rem 0.625rem;
      cursor: pointer;
      user-select: none;
      .label {
        flex: 1;
      }
      &:hover {
        background: $theme-palette-bg-secondary;
      }
      &.selected {
        background: $theme-palette-accent-hover;
      }
      input[type='checkbox'] {
        pointer-events: none;
      }
    }
    .empty {
      padding: 0.5rem 0.625rem;
      color: $theme-palette-text-muted;
    }
  }
}

@media (max-width: 640px) {
  .combobox {
    flex: 1 1 auto;
    margin-bottom: $spacing-xs;
    margin-right: 0;
    max-width: 100%;
    min-width: 8rem;
    order: 1;
    width: 100%;
  }
}
</style>
