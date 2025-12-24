<template>
  <div class="library-layout">
    <div class="library-container">
      <aside class="library-sidebar">
        <slot name="sidebar" />
      </aside>
      <div class="library-separator" aria-hidden="true"></div>
      <div class="library-main">
        <div class="library-header">
          <slot name="title" />
          <slot name="subtitle" />
        </div>
        <slot name="main" />
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.library-container {
  display: flex;
  height: 100vh;
  max-height: 100vh;
  max-width: 100vw;
  overflow: hidden;
}

.library-header {
  display: flex;
  align-items: center;
  gap: $spacing-md;
  padding: $spacing-lg 0;
  margin-left: 1rem;
}

.library-layout {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100vw;
}

.library-main {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 0;
}

.library-separator {
  width: 1px;
  height: 100vh;
  background: $color-gray-900;
  align-self: stretch;
}

.library-sidebar {
  width: 280px;
  min-width: 220px;
  max-width: 320px;
  background: transparent;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.library-subtitle {
  font-size: $font-size-lg;
  color: $color-gray-500;
}

.library-title {
  font-size: $font-size-3xl;
  font-weight: 700;
  margin: 0;
  color: $color-gray-700;

  @media (prefers-color-scheme: dark) {
    color: $color-gray-100;
  }
}
</style>
