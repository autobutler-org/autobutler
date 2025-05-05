// https://nuxt.com/docs/api/configuration/nuxt-config
import { defineNuxtConfig } from "nuxt/config";

export default defineNuxtConfig({
  modules: ["@nuxt/eslint", "@nuxtjs/tailwindcss"],

  app: {
    head: {
      title: "AutoButler - Your AI Assistant",
      link: [{ rel: "icon", type: "image/png", href: "/butler.png" }],
      meta: [
        { charset: "utf-8" },
        { name: "viewport", content: "width=device-width, initial-scale=1" },
        {
          name: "description",
          content:
            "Your intelligent AI assistant powered by advanced language models",
        },
      ],
    },
  },

  // CORS configuration for the server
  nitro: {
    routeRules: {
      "/api/**": {
        cors: true,
        headers: {
          "Access-Control-Allow-Methods": "GET,HEAD,PUT,PATCH,POST,DELETE",
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Headers": "Content-Type",
        },
      },
    },
  },

  compatibilityDate: "2025-04-01",
});
