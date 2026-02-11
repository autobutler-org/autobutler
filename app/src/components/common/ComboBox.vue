<template>
  <div class="combobox" @keydown.esc="close">
    <div class="control" @click="toggle" :class="{ open: isOpen }">
      <div class="tags" v-if="selected.length">
        <span class="tag" v-for="s in selected" :key="s.value">
          {{ s.label }}
          <button class="remove" @click.stop="deselect(s.value)">×</button>
        </span>
      </div>
      <input
        v-model="query"
        :placeholder="selected.length ? '' : placeholder"
        @focus="open"
        @input="onInput"
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

const selected = computed(() => {
  const vals = props.modelValue || [];
  return internal.value.filter((i) => vals.includes(i.value));
});

watch(
  () => props.items,
  (v) => (internal.value = v || []),
);

const placeholder = props.placeholder ?? 'Select...';

const open = () => {
  isOpen.value = true;
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
  if (!isOpen.value) isOpen.value = true;
};

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
  const root = (getCurrentInstance() as any)?.vnode?.el as
    | HTMLElement
    | undefined;
  if (!root) return;
  const handler = (e: MouseEvent) => {
    if (!root.contains(e.target as Node)) close();
  };
  document.addEventListener('click', handler);
});
</script>

<style scoped lang="scss">
.combobox {
  position: relative;
  width: 100%;
  max-width: 420px;
  font-family: inherit;
  .control {
    display: flex;
    align-items: center;
    gap: 8px;
    border: 1px solid var(--border, #cfcfcf);
    padding: 6px 8px;
    border-radius: 6px;
    background: var(--bg, #fff);
    cursor: text;
    &.open {
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
    }
    .tags {
      display: flex;
      gap: 6px;
      flex-wrap: wrap;
      .tag {
        display: inline-flex;
        align-items: center;
        background: #eef2ff;
        color: #1e293b;
        padding: 2px 6px;
        border-radius: 999px;
        font-size: 12px;
        .remove {
          background: transparent;
          border: 0;
          margin-left: 6px;
          cursor: pointer;
        }
      }
    }
    .input {
      flex: 1 1 auto;
      border: 0;
      outline: none;
      min-width: 36px;
      background: transparent;
    }
    .chev {
      background: transparent;
      border: none;
      cursor: pointer;
      padding: 2px 6px;
    }
  }

  .menu {
    position: absolute;
    z-index: 40;
    left: 0;
    right: 0;
    margin-top: 6px;
    border: 1px solid var(--border, #e5e7eb);
    background: white;
    border-radius: 8px;
    max-height: 220px;
    overflow: auto;
    list-style: none;
    padding: 6px 4px;
    li {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 10px;
      cursor: pointer;
      user-select: none;
      .label {
        flex: 1;
      }
      &:hover {
        background: #f8fafc;
      }
      &.selected {
        background: #eef2ff;
      }
      input[type='checkbox'] {
        pointer-events: none;
      }
    }
    .empty {
      padding: 8px 10px;
      color: #9ca3af;
    }
  }
}
</style>
