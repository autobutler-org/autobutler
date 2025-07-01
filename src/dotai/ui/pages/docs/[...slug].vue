<template>
  <div>
    <PageContainer>
      <!-- Mobile navigation bar -->
      <div class="mobile-nav-bar">
        <!-- Menu button -->
        <button
          class="hamburger-btn"
          aria-label="Toggle navigation"
          @click="toggleSidebar"
        >
          <div class="hamburger-icon">
            <span />
            <span />
            <span />
          </div>
          <span class="hamburger-label">Menu</span>
        </button>

        <!-- On this page dropdown button -->
        <button
          v-if="data && !data._error && data.body?.toc?.links?.length"
          class="page-nav-toggle"
          aria-label="Toggle page navigation"
          @click="togglePageNav"
        >
          <span>On this page</span>
          <svg
            class="chevron"
            :class="{ 'chevron-open': pageNavOpen }"
            width="16"
            height="16"
            viewBox="0 0 16 16"
          >
            <path
              d="M10 4l-4 4 4 4"
              stroke="currentColor"
              stroke-width="2"
              fill="none"
            />
          </svg>
        </button>
      </div>

      <div class="docs-layout">
        <div class="left-content-container">
          <aside
            class="sidebar"
            :class="{ 'sidebar-open': sidebarOpen }"
            aria-label="Documentation sidebar"
          >
            <nav aria-label="Documentation navigation">
              <ul>
                <li v-for="doc in sortedDocs" :key="doc.path">
                  <NuxtLink
                    :to="doc.path || '/docs'"
                    :class="{
                      'sidebar-active': isCurrentPath(doc.path || '/docs'),
                    }"
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
          />

          <main class="content">
            <article class="main-content">
              <!-- Loading indicator -->
              <div v-if="pending" class="loading-indicator">
                <div class="loading-spinner" />
                <p>Loading documentation...</p>
              </div>

              <!-- Document content -->
              <div v-else-if="data && !data._error" class="document-content">
                <ContentRenderer :value="data" />
              </div>

              <!-- Index page fallback with docs grid -->
              <div
                v-else-if="isIndexPage && (!data || data._error)"
                class="error-content"
              >
                <h1>Welcome to AutoButler Documentation</h1>
                <p>
                  Complete documentation for AutoButler automation platform.
                </p>
                <div class="docs-grid">
                  <NuxtLink
                    v-for="doc in sortedDocs"
                    :key="doc._path"
                    :to="doc._path"
                    class="doc-card"
                  >
                    <h3>{{ doc.navigation?.title || doc.title }}</h3>
                    <p>{{ doc.description }}</p>
                  </NuxtLink>
                </div>
              </div>

              <!-- Error state for other pages -->
              <div
                v-else-if="error || (data && data._error)"
                class="error-content"
              >
                <h1>Content Not Found</h1>
                <p>The requested documentation page could not be found.</p>
                <p v-if="data && data._error">{{ data.message }}</p>
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
                <p>
                  Welcome to the AutoButler documentation. Select a topic from
                  the sidebar to begin.
                </p>
              </div>
            </article>
          </main>
        </div>

        <!-- Right-side page navigation drawer -->
        <aside
          v-if="data && !data._error && data.body?.toc?.links?.length"
          class="page-nav-drawer"
          :class="{ 'page-nav-drawer-open': pageNavOpen }"
          aria-label="Page table of contents"
        >
          <div class="page-nav-drawer-content">
            <h4>On this page</h4>
            <nav class="toc-nav" aria-label="Page sections">
              <ul>
                <li v-for="link in data.body.toc.links" :key="link.id">
                  <a
                    :href="`#${link.id}`"
                    :class="`toc-link depth-${link.depth}`"
                    @click="closePageNav"
                  >
                    {{ link.text }}
                  </a>
                </li>
              </ul>
            </nav>
          </div>
        </aside>

        <!-- Overlay for page navigation drawer -->
        <div
          v-if="pageNavOpen"
          class="page-nav-overlay"
          @click="closePageNav"
        />

        <!-- Desktop page navigation -->
        <aside
          v-if="data && !data._error && data.body?.toc?.links?.length"
          class="page-nav desktop-only"
          aria-label="Page table of contents"
        >
          <div class="page-nav-content">
            <h4>On this page</h4>
            <nav class="toc-nav" aria-label="Page sections">
              <ul>
                <li v-for="link in data.body.toc.links" :key="link.id">
                  <a
                    :href="`#${link.id}`"
                    :class="`toc-link depth-${link.depth}`"
                  >
                    {{ link.text }}
                  </a>
                </li>
              </ul>
            </nav>
          </div>
        </aside>
      </div>
    </PageContainer>

    <!-- Reverse sticky footer for docs - moved outside PageContainer -->
    <ButlerFooter
      :show-on-bottom="true"
      custom-scroll-container=".content"
      :compact="true"
    />

    <!-- Mobile-only scroll to top button -->
    <button
      v-if="showScrollToTop"
      class="scroll-to-top-btn mobile-only"
      aria-label="Scroll to top"
      @click="scrollToTop"
    >
      <svg
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <path d="M18 15l-6-6-6 6" />
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
// Import the ButlerFooter component
import ButlerFooter from "~/components/ButlerFooter.vue";
import type { ParsedContent } from "@nuxt/content";
import {
  createNavigationHelpers,
  scrollHelpers,
  pathHelpers,
  setupDesktopScrolling,
} from "~/utils/docs/docsHelpers";

// Reactive state
const sidebarOpen = ref(false);
const pageNavOpen = ref(false);
const showScrollToTop = ref(false);

// Get route and params
const route = useRoute();

// Check if this is the index page (no slug or welcome slug)
const isIndexPage = computed(
  () =>
    route.path === "/docs" ||
    route.path === "/docs/" ||
    (route.params.slug && route.params.slug[0] === "welcome"),
);

// Use proper Nuxt data fetching without top-level awaits
const { data: allDocs } = await useAsyncData(route.path, () =>
  queryCollection("content").all(),
);

// Get current document based on route
const {
  data: currentData,
  error,
  pending,
} = await useAsyncData(`doc-${route.path || "/docs"}`, async () => {
  if (isIndexPage.value) {
    // For index page, try to get welcome content
    try {
      return await queryCollection("content").path("/docs/welcome").first();
    } catch (err) {
      // Return a serializable error object instead of the raw Error
      return {
        _error: true,
        message: err instanceof Error ? err.message : String(err),
        code: "CONTENT_NOT_FOUND",
      };
    }
  } else {
    // For other pages, get content based on slug
    try {
      return await queryCollection("content").path(route.path).first();
    } catch (err) {
      // Return a serializable error object instead of the raw Error
      return {
        _error: true,
        message: err instanceof Error ? err.message : String(err),
        code: "CONTENT_NOT_FOUND",
      };
    }
  }
});

// Use the data from composables - ParsedContent is auto-imported by Nuxt Content
const data = currentData as Ref<ParsedContent | null>;

// Computed properties
const sortedDocs = computed(() =>
  ((allDocs.value || []) as ParsedContent[])
    .slice()
    .sort(
      (a, b) => (a.navigation?.order || 999) - (b.navigation?.order || 999),
    ),
);

// Create navigation helpers
const { toggleSidebar, closeSidebar, togglePageNav, closePageNav } =
  createNavigationHelpers(sidebarOpen, pageNavOpen);

// Path helper
const isCurrentPath = (path: string) =>
  pathHelpers.isCurrentPath(path, route.path);

// Scroll functions
const { scrollToTop } = scrollHelpers;
const handleMobileScroll = scrollHelpers.handleMobileScroll(showScrollToTop);

// Setup desktop scrolling and mobile scroll detection
onMounted(async () => {
  const isDesktop = () => window.innerWidth >= 1024;

  if (!isDesktop()) {
    // On mobile/tablet, let browser handle scrolling normally
    // But add scroll listener for scroll-to-top button
    window.addEventListener("scroll", handleMobileScroll);

    // Cleanup on unmount for mobile
    onUnmounted(() => {
      window.removeEventListener("scroll", handleMobileScroll);
    });
    return;
  }

  // Setup desktop scrolling
  const cleanup = await setupDesktopScrolling();
  onUnmounted(cleanup);
});

// SEO
useSeoMeta({
  title: computed(() => {
    const currentData = unref(data);
    if (isIndexPage.value) {
      return "AutoButler Documentation";
    }
    return currentData?.title
      ? `${currentData.title} - AutoButler Docs`
      : "AutoButler Documentation";
  }),
  description: computed(() => {
    const currentData = unref(data);
    if (isIndexPage.value) {
      return "Welcome to AutoButler - your intelligent automation platform";
    }
    return (
      currentData?.description ||
      "Complete documentation for AutoButler automation platform"
    );
  }),
});
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
  grid-template-columns: 1fr 250px;
  gap: 2rem;
  height: calc(100vh - 8rem);
  overflow: hidden;
}

.left-content-container {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 2rem;
  overflow: hidden;
  height: 100%;
}

.sidebar {
  border-right: 1px solid rgba(255, 255, 255, 0.1);
  padding-right: 2rem;
  overflow-y: auto;
  height: 100%;
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
  overflow-y: auto;
  height: calc(100vh - 12rem);
  scrollbar-width: thin;
  scrollbar-color: rgba(0, 255, 170, 0.3) transparent;
  margin-bottom: 4rem;
  border-radius: 0.5rem;
  background: rgba(0, 0, 0, 0.1);
  padding-bottom: 100px;
}

/* Custom scrollbar for WebKit browsers */
.content::-webkit-scrollbar {
  width: 6px;
}

.content::-webkit-scrollbar-track {
  background: transparent;
}

.content::-webkit-scrollbar-thumb {
  background: rgba(0, 255, 170, 0.3);
  border-radius: 3px;
  transition: all 0.2s ease;
}

.content::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 255, 170, 0.6);
  box-shadow: 0 0 8px rgba(0, 255, 170, 0.3);
}

.main-content {
  min-width: 0;
  padding: 1rem;
  padding-bottom: 2rem; /* Extra padding for footer space */
}

.page-nav {
  position: static;
  height: 100%;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.2) transparent;
  padding: 2rem 0;
}

.page-nav::-webkit-scrollbar {
  width: 4px;
}

.page-nav::-webkit-scrollbar-track {
  background: transparent;
}

.page-nav::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 2px;
}

.page-nav::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

.page-nav-content {
  border-left: 2px solid rgba(255, 255, 255, 0.1);
  padding-left: 1rem;
}

.page-nav-content h4 {
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
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.loading-indicator p {
  color: rgba(255, 255, 255, 0.7);
  margin: 0;
}

/* Content styles */
.document-content,
.error-content {
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

/* Docs grid styles for index page fallback */
.docs-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-top: 2rem;
}

.doc-card {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 0.75rem;
  padding: 1.5rem;
  text-decoration: none;
  transition: all 0.3s ease;
  display: block;
}

.doc-card:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(0, 255, 170, 0.3);
  transform: translateY(-2px);
  box-shadow: 0 10px 25px rgba(0, 255, 170, 0.1);
}

.doc-card h3 {
  font-size: 1.375rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
  color: #fff;
}

.doc-card p {
  color: rgba(255, 255, 255, 0.7);
  line-height: 1.6;
  margin: 0;
}

/* Mobile responsive styles */
@media (max-width: 1024px) {
  .mobile-nav-bar {
    display: flex;
  }

  .docs-layout {
    grid-template-columns: 1fr;
    height: auto;
  }

  .left-content-container {
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
    height: auto;
  }

  .left-content-container {
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
    height: auto;
    overflow-y: visible;
  }

  .main-content {
    padding: 0;
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

  .docs-grid {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .doc-card {
    padding: 1.25rem;
  }
}

/* Custom TOC navigation styles */
.toc-nav ul {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.toc-nav li {
  margin: 0;
}

.toc-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: rgba(255, 255, 255, 0.7);
  text-decoration: none;
  font-size: 0.875rem;
  line-height: 1.4;
  padding: 0.375rem 0.5rem;
  border-radius: 0.25rem;
  transition: all 0.2s ease;
  position: relative;
}

.toc-link:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.05);
}

.toc-link:before {
  content: "";
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.3);
  flex-shrink: 0;
  transition: all 0.2s ease;
}

.toc-link:hover:before {
  background: rgba(0, 255, 170, 0.8);
  box-shadow: 0 0 8px rgba(0, 255, 170, 0.3);
}

/* Nested heading levels */
.toc-link.depth-3 {
  font-size: 0.8rem;
  color: rgba(255, 255, 255, 0.6);
  padding: 0.25rem 0.5rem 0.25rem 1rem;
}

.toc-link.depth-3:before {
  width: 4px;
  height: 4px;
  background: rgba(255, 255, 255, 0.2);
}

.toc-link.depth-4 {
  font-size: 0.75rem;
  color: rgba(255, 255, 255, 0.5);
  padding: 0.25rem 0.5rem 0.25rem 1.5rem;
}

.toc-link.depth-4:before {
  width: 3px;
  height: 3px;
  background: rgba(255, 255, 255, 0.15);
}

/* H2 level links get special styling */
.toc-link.depth-2 {
  font-weight: 600;
  color: rgba(255, 255, 255, 0.9);
  font-size: 0.9rem;
}

.toc-link.depth-2:before {
  width: 8px;
  height: 8px;
  background: rgba(0, 255, 170, 0.6);
}

/* Mobile scroll-to-top button */
.scroll-to-top-btn {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  width: 3rem;
  height: 3rem;
  background: rgba(0, 255, 170, 0.9);
  border: none;
  border-radius: 50%;
  color: #000;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  box-shadow: 0 4px 12px rgba(0, 255, 170, 0.3);
  transition: all 0.3s ease;
  backdrop-filter: blur(10px);
}

.scroll-to-top-btn:hover {
  background: rgba(0, 255, 170, 1);
  box-shadow: 0 6px 20px rgba(0, 255, 170, 0.4);
  transform: translateY(-2px);
}

.scroll-to-top-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(0, 255, 170, 0.3);
}

/* Hide on desktop */
@media (min-width: 1024px) {
  .mobile-only {
    display: none !important;
  }
}
</style>
