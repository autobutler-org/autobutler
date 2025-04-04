<template>
  <div class="min-h-screen bg-gradient-to-br from-gray-900 to-gray-800">
    <div class="relative min-h-screen">
      <!-- Gradient overlays -->
      <div class="absolute inset-0">
        <div
          class="absolute top-0 left-0 w-1/3 h-1/3 bg-blue-500/10 rounded-full blur-3xl"
        ></div>
        <div
          class="absolute bottom-0 right-0 w-1/3 h-1/3 bg-purple-500/10 rounded-full blur-3xl"
        ></div>
      </div>

      <!-- Content -->
      <div class="relative z-10">
        <slot />
      </div>
    </div>
  </div>
</template>

<style>
html,
body {
  margin: 0;
  padding: 0;
  min-height: 100vh;
  min-width: 100vw;
  overflow-x: hidden;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

#__nuxt {
  min-height: 100vh;
  min-width: 100vw;
}
</style>
