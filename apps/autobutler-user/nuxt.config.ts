export default defineNuxtConfig({
  modules: ['@nuxtjs/tailwindcss'],

  app: {
    head: {
      title: 'AutoButler - Your AI Assistant',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Your intelligent AI assistant powered by advanced language models' }
      ]
    }
  },

  compatibilityDate: '2025-04-01'
})