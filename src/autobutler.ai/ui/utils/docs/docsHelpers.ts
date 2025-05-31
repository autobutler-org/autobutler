import type { Ref } from 'vue'

// Navigation helper functions
export const createNavigationHelpers = (
  sidebarOpen: Ref<boolean>,
  pageNavOpen: Ref<boolean>
) => ({
  toggleSidebar: () => {
    sidebarOpen.value = !sidebarOpen.value;
  },

  closeSidebar: () => {
    sidebarOpen.value = false;
  },

  togglePageNav: () => {
    pageNavOpen.value = !pageNavOpen.value;
  },

  closePageNav: () => {
    pageNavOpen.value = false;
  },
});

// Scroll utility functions
export const scrollHelpers = {
  scrollToTop: () => {
    window.scrollTo({
      top: 0,
      behavior: "smooth",
    });
  },

  handleMobileScroll: (showScrollToTop: Ref<boolean>) => () => {
    const isDesktop = window.innerWidth >= 1024;
    if (isDesktop) return;

    // Show button when scrolled down more than 300px
    showScrollToTop.value = window.scrollY > 300;
  },
};

// Desktop scroll handling functions
export const desktopScrollHandlers = {
  ensureContentReady: (): Promise<void> => {
    return new Promise((resolve) => {
      const checkContent = () => {
        const contentArea = document.querySelector(".content");
        if (!contentArea || contentArea.scrollHeight <= contentArea.clientHeight) {
          setTimeout(checkContent, 100);
          return;
        }

        // Only prevent page scrolling once we confirm content area is ready
        document.body.style.overflow = "hidden";
        document.documentElement.style.overflow = "hidden";
        resolve();
      };
      checkContent();
    });
  },

  handleWheel: (e: WheelEvent) => {
    e.preventDefault();
    const contentArea = document.querySelector(".content");
    if (contentArea) {
      contentArea.scrollTop += e.deltaY;
    }
  },

  handleKeydown: (e: KeyboardEvent) => {
    const contentArea = document.querySelector(".content");
    if (!contentArea) return;

    const keyActions: Record<string, () => void> = {
      ArrowDown: () => {
        e.preventDefault();
        contentArea.scrollTop += 40;
      },
      ArrowUp: () => {
        e.preventDefault();
        contentArea.scrollTop -= 40;
      },
      PageDown: () => {
        e.preventDefault();
        contentArea.scrollTop += contentArea.clientHeight * 0.8;
      },
      PageUp: () => {
        e.preventDefault();
        contentArea.scrollTop -= contentArea.clientHeight * 0.8;
      },
      Home: () => {
        if (e.ctrlKey) {
          e.preventDefault();
          contentArea.scrollTop = 0;
        }
      },
      End: () => {
        if (e.ctrlKey) {
          e.preventDefault();
          contentArea.scrollTop = contentArea.scrollHeight;
        }
      },
    };

    const action = keyActions[e.key];
    if (action) action();
  },

  handleAnchorClick: (e: Event) => {
    const target = e.target as HTMLElement;
    if (
      !target ||
      target.tagName !== "A" ||
      !(target as HTMLAnchorElement).getAttribute("href")?.startsWith("#")
    ) {
      return;
    }

    e.preventDefault();
    const targetId = (target as HTMLAnchorElement)
      .getAttribute("href")
      ?.substring(1);
    const targetElement = targetId ? document.getElementById(targetId) : null;
    const contentArea = document.querySelector(".content");

    if (targetElement && contentArea) {
      const contentRect = contentArea.getBoundingClientRect();
      const targetRect = targetElement.getBoundingClientRect();
      const scrollOffset =
        targetRect.top - contentRect.top + contentArea.scrollTop - 20;

      contentArea.scrollTo({
        top: scrollOffset,
        behavior: "smooth",
      });
    }
  },
};

// Path utility functions
export const pathHelpers = {
  isCurrentPath: (path: string, currentRoute: string) => {
    // For the welcome page, consider both /docs and /docs/welcome as current
    if (
      path === "/docs/welcome" &&
      (currentRoute === "/docs" || currentRoute === "/docs/")
    ) {
      return true;
    }
    return currentRoute === path;
  },
};

// Desktop scroll setup function
export const setupDesktopScrolling = async () => {
  const isDesktop = () => window.innerWidth >= 1024;
  if (!isDesktop()) return () => {}; // Return empty cleanup for mobile

  await desktopScrollHandlers.ensureContentReady();

  // Add event listeners
  window.addEventListener("wheel", desktopScrollHandlers.handleWheel, { passive: false });
  window.addEventListener("keydown", desktopScrollHandlers.handleKeydown);
  document.addEventListener("click", desktopScrollHandlers.handleAnchorClick);

  // Return cleanup function
  return () => {
    document.body.style.overflow = "";
    document.documentElement.style.overflow = "";
    window.removeEventListener("wheel", desktopScrollHandlers.handleWheel);
    window.removeEventListener("keydown", desktopScrollHandlers.handleKeydown);
    document.removeEventListener("click", desktopScrollHandlers.handleAnchorClick);
  };
}; 