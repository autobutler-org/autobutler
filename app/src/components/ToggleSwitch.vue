<template>
  <label class="toggle-switch">
    <input
      type="checkbox"
      :checked="modelValue"
      @change="
        $emit('update:modelValue', ($event.target as HTMLInputElement).checked)
      "
    />
    <span class="slider"></span>
  </label>
</template>

<script lang="ts" setup>
defineProps<{ modelValue: boolean }>();
defineEmits(['update:modelValue']);
</script>

<style lang="scss" scoped>
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 2.5rem;
  height: 1.4rem;
}
.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}
.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: $theme-palette-border;
  transition: 0.3s;
  border-radius: 1.4rem;
}
.slider:before {
  position: absolute;
  content: '';
  height: 1.1rem;
  width: 1.1rem;
  left: 0.15rem;
  bottom: 0.15rem;
  background: $theme-palette-bg-primary;
  transition: 0.3s;
  border-radius: 50%;
}
input:checked + .slider {
  background: $theme-palette-accent;
}
input:checked + .slider:before {
  transform: translateX(1.1rem);
  background: $theme-palette-bg-primary; // or white for contrast
}
</style>
