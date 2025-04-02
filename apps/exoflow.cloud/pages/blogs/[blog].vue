<template>
  <div class="blog-post">
    <h1>{{ blog.title }}</h1>
    <ContentDoc />
  </div>
</template>

<script setup>
import { useRoute } from 'vue-router'

const route = useRoute()
const { data: blog } = queryContent(`blog/${route.params.blog}`)

definePageMeta({
  layout: 'default'
})
</script>
