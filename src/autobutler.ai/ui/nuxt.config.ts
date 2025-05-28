// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: "2024-11-01",
  devtools: { enabled: true },
  modules: ["@nuxt/eslint", "@nuxt/content"],
  ssr: true,
  css: [
    '~/assets/css/content.css'
  ],
  content: {
    // Configure content module
    highlight: {
      theme: {
        default: 'github-dark',
        dark: 'github-dark',
        light: 'github-light'
      },
      preload: ['json', 'js', 'ts', 'html', 'css', 'vue', 'bash', 'yaml']
    },
    markdown: {
      // Enable anchor links for headings
      anchorLinks: true,
      // Enable table of contents
      toc: {
        depth: 3,
        searchDepth: 3
      }
    }
  }
});
