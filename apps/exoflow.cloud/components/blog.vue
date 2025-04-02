<template>
    <div class="docs-page">
      <ContentDoc :path="`../../content/blog/${route.params.id}`"/>
    </div>
  </template>
  
  <style scoped>
  .docs-page {
    width: 100%;
    max-width: 800px;
    margin: 0 auto;
    padding: 1rem;
  }
  
  /* Style markdown content */
  :deep(h1) {
    color: var(--primary-color);
    margin-bottom: 2rem;
  }
  
  :deep(h2) {
    color: var(--primary-light);
    margin: 2rem 0 1rem;
  }
  
  :deep(pre) {
    background: #242424;
    padding: 1rem;
    border-radius: 4px;
    overflow-x: auto;
  }
  
  :deep(code) {
    background: #242424;
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
  }
  
  :deep(ul) {
    padding-left: 1.5rem;
  }
  
  :deep(li) {
    margin: 0.5rem 0;
  }
  </style>