<template>
  <th
    :class="['file-table-header-cell', 'file-table-header-cell--sortable', alignClass]"
    @click="handleToggleSort(header)"
  >
    <span class="sort-button">
      <span>{{ displayHeader }}</span>
      <span class="sort-arrows">
        <SortArrowAscend v-if="activeSortColumn === header && sortDirection === 'asc'" />
        <SortArrowDescend v-else-if="activeSortColumn === header && sortDirection === 'desc'" />
        <SortArrowNeutral v-else />
      </span>
    </span>
  </th>
</template>

<script setup lang="ts">
import SortArrowAscend from '../icons/SortArrowAscend.vue';
import SortArrowDescend from '../icons/SortArrowDescend.vue';
import SortArrowNeutral from '../icons/SortArrowNeutral.vue';

export type HeaderAlignDirection = 'left' | 'right'
export type SortColumn = 'name' | 'size' | null
export type SortDirection = 'asc' | 'desc'

const props = defineProps<{
  header: SortColumn
  activeSortColumn: SortColumn
  sortDirection: SortDirection
  alignDirection?: HeaderAlignDirection
}>()
const emit = defineEmits<{
  'toggle:sort': [column: SortColumn]
}>()

const handleToggleSort = (header: SortColumn) => {
  emit('toggle:sort', header)
}

const sortColumnToHeaderCase = (str: SortColumn) =>
  str === null ? '' : str.charAt(0).toUpperCase() + str.slice(1)

const displayHeader = sortColumnToHeaderCase(props.header)
const alignClass =
  props.alignDirection === 'left' ? 'file-table-header-cell--left' : 'file-table-header-cell--right'
</script>

<style lang="scss" scoped>
.file-table-header-cell {
  height: 3rem;
  padding: 0 $spacing-sm;
  font-weight: 600;
  color: $color-gray-700;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-300;
  }

  &--left {
    text-align: left;
  }

  &--right {
    text-align: right;
    width: 6rem;
  }

  &--sortable {
    cursor: pointer;
    user-select: none;

    &:hover {
      background-color: $color-gray-100;

      @media (prefers-color-scheme: dark) {
        background-color: $color-gray-800;
      }
    }
  }
}

.sort-arrow {
  width: 16px;
  height: 16px;
  color: $color-gray-400;

  &--active {
    color: $color-gray-700;

    @media (prefers-color-scheme: dark) {
      color: $color-gray-300;
    }
  }
}

.sort-arrows {
  display: inline-flex;
  align-items: center;
}

.sort-button {
  display: inline-flex;
  align-items: center;
  gap: $spacing-xs;
}
</style>
