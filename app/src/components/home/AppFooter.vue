<template>
  <footer class="app-footer">
    <p class="footer-text">Autobutler LLC.</p>
    <a
      href="https://github.com/autobutler-org/autobutler"
      target="_blank"
      rel="noopener noreferrer"
    >
      <img src="/img/github/github-mark-white.svg" alt="GitHub logo" />
    </a>
  </footer>
</template>

<style lang="scss" scoped>
.app-footer {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: $spacing-lg;
  padding: $spacing-xl;
  margin-top: auto;
  color: $theme-palette-text-muted;

  img {
    width: 24px;
    height: 24px;
    opacity: 0.7;
    transition: opacity 0.2s ease;
  }

  a:hover img {
    opacity: 1;
  }
}

.footer-text {
  font-size: $theme-font-size-sm;
  margin: 0;
}
</style>
