// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: "2024-11-01",
  devtools: { enabled: true },
  modules: ["@nuxt/eslint", "@nuxt/content"],
  ssr: true,
  css: ["~/assets/css/content.css"],
  app: {
    head: {
      link: [
        {
          rel: "icon",
          type: "image/png",
          sizes: "32x32",
          href: "/favicon.png",
        },
        {
          rel: "icon",
          type: "image/png",
          sizes: "16x16",
          href: "/favicon-16x16.png",
        },
        { rel: "shortcut icon", href: "/favicon.ico" },
      ],
    },
  },
  content: {
    // Configure content module for better TOC generation
    highlight: {
      theme: {
        default: "github-dark",
        dark: "github-dark",
        light: "github-light",
      },
      preload: [
        "json",
        "js",
        "ts",
        "html",
        "css",
        "vue",
        "bash",
        "yaml",
        "markdown",
      ],
    },
    markdown: {
      // Enable anchor links for headings
      anchorLinks: {
        depth: 6,
        exclude: [],
      },
      // Enable comprehensive table of contents
      toc: {
        depth: 5,
        searchDepth: 6,
      },
      // Generate IDs for all headings
      remarkPlugins: [],
      rehypePlugins: [],
    },
  },
});
