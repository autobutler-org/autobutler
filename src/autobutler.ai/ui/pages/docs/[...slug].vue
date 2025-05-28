<template>
  <PageContainer>
    <!-- Mobile navigation bar -->
    <div class="mobile-nav-bar">
      <!-- Menu button -->
      <button 
        class="hamburger-btn"
        @click="toggleSidebar"
        aria-label="Toggle navigation"
      >
        <div class="hamburger-icon">
          <span></span>
          <span></span>
          <span></span>
        </div>
        <span class="hamburger-label">Menu</span>
      </button>

      <!-- On this page dropdown button -->
      <button 
        class="page-nav-toggle"
        @click="togglePageNav"
        aria-label="Toggle page navigation"
        v-if="data?.body?.toc?.links?.length"
      >
        <span>On this page</span>
        <svg 
          class="chevron" 
          :class="{ 'chevron-open': pageNavOpen }"
          width="16" 
          height="16" 
          viewBox="0 0 16 16"
        >
          <path d="M10 4l-4 4 4 4" stroke="currentColor" stroke-width="2" fill="none"/>
        </svg>
      </button>
    </div>

    <div class="docs-layout">
      <aside class="sidebar" :class="{ 'sidebar-open': sidebarOpen }">
        <nav>
          <ul>
            <li v-for="doc in sortedDocs" :key="doc._path">
              <NuxtLink 
                :to="doc._path"
                :class="{ 'sidebar-active': isCurrentPath(doc._path) }"
                @click="closeSidebar"
              >
                {{ doc.navigation?.title || doc.title }}
              </NuxtLink>
            </li>
          </ul>
        </nav>
      </aside>
      
      <!-- Overlay for mobile sidebar -->
      <div 
        v-if="sidebarOpen" 
        class="sidebar-overlay"
        @click="closeSidebar"
      ></div>
      
      <!-- Right-side page navigation drawer -->
      <aside 
        class="page-nav-drawer" 
        :class="{ 'page-nav-drawer-open': pageNavOpen }"
        v-if="data?.body?.toc?.links?.length"
      >
        <div class="page-nav-drawer-content">
          <h4>On this page</h4>
          <ContentNavigation :links="data.body.toc.links" />
        </div>
      </aside>
      
      <!-- Overlay for page navigation drawer -->
      <div 
        v-if="pageNavOpen" 
        class="page-nav-overlay"
        @click="closePageNav"
      ></div>
      
      <main class="content">
        <div class="content-wrapper">
          <article class="main-content">
            <!-- Loading indicator -->
            <div v-if="pending" class="loading-indicator">
              <div class="loading-spinner"></div>
              <p>Loading documentation...</p>
            </div>
            
            <!-- Document content -->
            <div v-else-if="data" class="document-content">
              <ContentRenderer :value="data" />
            </div>
            
            <!-- Error state -->
            <div v-else-if="error" class="error-content">
              <h1>Content Not Found</h1>
              <p>The requested documentation page could not be found.</p>
              <p>Available pages:</p>
              <ul>
                <li v-for="doc in sortedDocs" :key="doc._path">
                  <NuxtLink :to="doc._path">{{ doc.title }}</NuxtLink>
                </li>
              </ul>
            </div>
            
            <!-- Default fallback -->
            <div v-else class="error-content">
              <h1>Documentation</h1>
              <p>Welcome to the AutoButler documentation. Select a topic from the sidebar to begin.</p>
            </div>
          </article>
          
          <!-- Desktop page navigation -->
          <aside 
            class="page-nav desktop-only" 
            v-if="data?.body?.toc?.links?.length"
          >
            <div class="page-nav-content">
              <h4>On this page</h4>
              <ContentNavigation :links="data.body.toc.links" />
            </div>
          </aside>
        </div>
      </main>
    </div>
  </PageContainer>
</template>

<script setup>
// Reactive state
const sidebarOpen = ref(false)
const pageNavOpen = ref(false)

// Get route and params
const route = useRoute()

// Fetch all documentation files
const { data: allDocs } = await queryContent('/docs').find()

// Get current document
const { data, pending, error } = await queryContent()
  .where({ _path: route.path })
  .findOne()

// Computed properties
const sortedDocs = computed(() => 
  allDocs?.sort((a, b) => (a.navigation?.order || 999) - (b.navigation?.order || 999)) || []
)

const isCurrentPath = (path) => route.path === path

// Navigation functions
const toggleSidebar = () => {
  sidebarOpen.value = !sidebarOpen.value
}

const closeSidebar = () => {
  sidebarOpen.value = false
}

const togglePageNav = () => {
  pageNavOpen.value = !pageNavOpen.value
}

const closePageNav = () => {
  pageNavOpen.value = false
}

// SEO
useSeoMeta({
  title: computed(() => data.value?.title ? `${data.value.title} - AutoButler Docs` : 'AutoButler Documentation'),
  description: computed(() => data.value?.description || 'Complete documentation for AutoButler automation platform'),
})
</script>

<style scoped>
.mobile-nav-bar {
  display: none;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(10px);
  position: sticky;
  top: 0;
  z-index: 100;
  gap: 1rem;
}

.hamburger-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0.5rem 0.75rem;
  color: #fff;
  font-size: 0.9rem;
  border-radius: 0.375rem;
  transition: background-color 0.2s ease;
}

.hamburger-btn:hover {
  background: rgba(255, 255, 255, 0.05);
}

.hamburger-icon {
  display: flex;
  flex-direction: column;
  width: 1.25rem;
  height: 1rem;
  justify-content: space-between;
}

.hamburger-icon span {
  display: block;
  height: 2px;
  width: 100%;
  background: #fff;
  transition: all 0.3s ease;
}

.hamburger-label {
  font-weight: 500;
}

.page-nav-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.375rem;
  cursor: pointer;
  padding: 0.5rem 0.75rem;
  color: #fff;
  font-size: 0.9rem;
  font-weight: 500;
  gap: 0.5rem;
  transition: all 0.2s ease;
  min-width: 140px;
}

.page-nav-toggle:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.2);
}

.chevron {
  transition: transform 0.2s ease;
  color: rgba(255, 255, 255, 0.7);
  flex-shrink: 0;
}

.chevron-open {
  transform: rotate(180deg);
}

.docs-layout {
  display: grid;
  grid-template-columns: 250px 1fr;
  gap: 2rem;
}

.sidebar {
  border-right: 1px solid rgba(255, 255, 255, 0.1);
  padding-right: 2rem;
}

.sidebar ul {
  list-style: none;
  padding: 0;
}

.sidebar a {
  display: block;
  padding: 0.5rem 0;
  color: rgba(255, 255, 255, 0.8);
  text-decoration: none;
  transition: all 0.3s ease;
}

.sidebar a:hover {
  color: #fff;
  padding-left: 0.5rem;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.1),
    rgba(0, 187, 255, 0.1)
  );
}

.sidebar a.sidebar-active {
  color: #fff;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.2),
    rgba(0, 187, 255, 0.2)
  );
  border-left: 3px solid rgba(0, 255, 170, 0.8);
  padding-left: 0.5rem;
  font-weight: 600;
}

.content {
  min-width: 0;
}

.content-wrapper {
  display: grid;
  grid-template-columns: 1fr 200px;
  gap: 3rem;
}

.main-content {
  min-width: 0;
}

.page-nav {
  position: sticky;
  top: 2rem;
  height: fit-content;
}

.page-nav-content {
  border-left: 2px solid rgba(255, 255, 255, 0.1);
  padding-left: 1rem;
}

.page-nav h4 {
  color: rgba(255, 255, 255, 0.6);
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 0.75rem 0;
  font-weight: 600;
}

.page-nav-drawer {
  position: fixed;
  top: 0;
  right: 0;
  height: 100vh;
  width: 280px;
  background: rgba(20, 20, 20, 0.95);
  backdrop-filter: blur(10px);
  border-left: 1px solid rgba(255, 255, 255, 0.1);
  padding: 2rem;
  transform: translateX(100%);
  transition: transform 0.3s ease;
  z-index: 1000;
  overflow-y: auto;
  display: none;
}

.page-nav-drawer-open {
  transform: translateX(0);
}

.page-nav-drawer-content h4 {
  color: rgba(255, 255, 255, 0.6);
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 1rem 0;
  font-weight: 600;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding-bottom: 0.75rem;
}

.page-nav-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.5);
  z-index: 999;
  display: none;
}

.desktop-only {
  display: block;
}

.sidebar-overlay {
  display: none;
}

/* Loading indicator styles */
.loading-indicator {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem;
  text-align: center;
}

.loading-spinner {
  width: 2rem;
  height: 2rem;
  border: 3px solid rgba(255, 255, 255, 0.1);
  border-top: 3px solid rgba(0, 255, 170, 0.8);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-indicator p {
  color: rgba(255, 255, 255, 0.7);
  margin: 0;
}

/* Content styles */
.document-content, .error-content {
  line-height: 1.6;
}

.error-content ul {
  list-style: none;
  padding: 0;
}

.error-content li {
  margin: 0.5rem 0;
}

.error-content a {
  color: rgba(0, 187, 255, 0.9);
  text-decoration: none;
  transition: color 0.2s ease;
}

.error-content a:hover {
  color: rgba(0, 255, 170, 0.9);
  text-decoration: underline;
}

/* Mobile styles */
@media (max-width: 1024px) {
  .mobile-nav-bar {
    display: flex;
  }
  
  .content-wrapper {
    grid-template-columns: 1fr;
    gap: 2rem;
  }
  
  .desktop-only {
    display: none;
  }
  
  .page-nav-drawer {
    display: block;
  }
  
  .page-nav-overlay {
    display: block;
  }
}

@media (max-width: 768px) {
  .docs-layout {
    grid-template-columns: 1fr;
    gap: 0;
  }
  
  .sidebar {
    position: fixed;
    top: 0;
    left: 0;
    height: 100vh;
    width: 250px;
    background: rgba(20, 20, 20, 0.95);
    backdrop-filter: blur(10px);
    border-right: 1px solid rgba(255, 255, 255, 0.1);
    padding: 4rem 2rem 2rem 2rem;
    transform: translateX(-100%);
    transition: transform 0.3s ease;
    z-index: 1000;
    overflow-y: auto;
  }
  
  .sidebar-open {
    transform: translateX(0);
  }
  
  .sidebar-overlay {
    display: block;
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.5);
    z-index: 999;
  }
  
  .content {
    padding: 1rem;
  }
  
  .content-wrapper {
    gap: 1.5rem;
  }
}

@media (max-width: 480px) {
  .mobile-nav-bar {
    padding: 0.75rem;
    gap: 0.5rem;
  }
  
  .page-nav-toggle {
    min-width: 120px;
    padding: 0.5rem;
    font-size: 0.85rem;
  }
  
  .hamburger-btn {
    padding: 0.5rem;
    font-size: 0.85rem;
  }
}
</style> 