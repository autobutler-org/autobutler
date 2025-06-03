<template>
  <footer
    v-show="isVisible"
    class="footer"
    :class="{
      'footer-visible': isVisible,
      'footer-compact': compact,
    }"
  >
    <div class="gradient-overlay" />
    <div class="footer-content">
      <div class="footer-info">
        <p class="copyright">
          © {{ currentYear }} AutoButler. All rights reserved.
        </p>
        <p class="version">{{ displayVersion }}</p>
      </div>
      <div class="footer-links">
        <a href="/privacy">Privacy</a>
        <a href="/terms">Terms</a>
        <a href="/contact">Contact</a>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, nextTick } from "vue";
import { useVersion } from "~/composables/useVersion";

// Props to control visibility
interface Props {
  showOnBottom?: boolean;
  customScrollContainer?: string;
  compact?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  showOnBottom: false,
  customScrollContainer: "",
  compact: false,
});

const { displayVersion } = useVersion();
const currentYear = computed(() => new Date().getFullYear());
const isVisible = ref(!props.showOnBottom);

// Handle scroll detection for reverse sticky behavior
let scrollHandler: ((e: Event) => void) | null = null;

onMounted(() => {
  if (props.showOnBottom) {
    isVisible.value = false;

    const handleScroll = (e: Event) => {
      let target: HTMLElement;

      // If we have a custom scroll container, get it directly
      if (props.customScrollContainer) {
        target = e.target as HTMLElement;
        console.log(
          "Scroll event on custom container:",
          props.customScrollContainer,
        );
        console.log("Target element:", target);
      } else {
        // For window scroll, use document.documentElement
        target = document.documentElement;
        console.log("Scroll event on window");
      }

      // Check if we're at the bottom of the scroll area
      const scrollTop = target.scrollTop;
      const scrollHeight = target.scrollHeight;
      const clientHeight = target.clientHeight;

      const distanceFromBottom = scrollHeight - clientHeight - scrollTop;
      const triggerDistance = 100; // Start showing 100px from bottom

      // Calculate progress (0 to 1) as user approaches bottom
      const progress = Math.max(
        0,
        Math.min(1, (triggerDistance - distanceFromBottom) / triggerDistance),
      );
      const isAtBottom = distanceFromBottom < triggerDistance;

      console.log("Scroll debug:", {
        scrollTop,
        scrollHeight,
        clientHeight,
        distanceFromBottom,
        progress,
        isAtBottom,
        currentVisible: isVisible.value,
      });

      // Update visibility and progress
      isVisible.value = isAtBottom;

      // Apply progressive transform for smooth appearance
      if (isAtBottom) {
        const footer = document.querySelector(".footer") as HTMLElement;
        if (footer) {
          const translateY = (1 - progress) * 100;
          footer.style.transform = `translateY(${translateY}%)`;
        }
      }
    };

    scrollHandler = handleScroll;

    // Use custom scroll container if provided, otherwise use window
    if (props.customScrollContainer) {
      // Wait for next tick to ensure DOM is ready
      nextTick(() => {
        const container = document.querySelector(props.customScrollContainer);
        console.log(
          "Looking for container:",
          props.customScrollContainer,
          "Found:",
          container,
        );
        if (container) {
          container.addEventListener("scroll", handleScroll);
          console.log("Added scroll listener to:", container);
        } else {
          console.warn(
            `Custom scroll container "${props.customScrollContainer}" not found`,
          );
        }
      });
    } else {
      window.addEventListener("scroll", handleScroll);
    }
  }
});

onUnmounted(() => {
  if (scrollHandler) {
    if (props.customScrollContainer) {
      const container = document.querySelector(props.customScrollContainer);
      if (container) {
        container.removeEventListener("scroll", scrollHandler);
      }
    } else {
      window.removeEventListener("scroll", scrollHandler);
    }
    scrollHandler = null;
  }
});
</script>

<style scoped>
.footer {
  width: 100%;
  padding: 1.5rem 0;
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: rgba(28, 32, 34, 0.98);
  backdrop-filter: blur(12px);
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  z-index: 1000;
  transform: translateY(100%);
  box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.15);
}

.footer-visible {
  /* Transform is now handled by JavaScript for smooth scroll effect */
}

.footer-compact {
  padding: 0.75rem 0; /* Half the normal padding */
}

.footer-compact .footer-content {
  flex-direction: row;
  align-items: center;
  gap: 1rem;
}

.footer-compact .footer-info {
  flex-direction: row;
  align-items: center;
  gap: 1rem;
}

.footer-compact .copyright {
  font-size: 0.8rem;
  margin: 0;
}

.footer-compact .version {
  font-size: 0.7rem;
  margin: 0;
}

.footer-compact .footer-links {
  gap: 1rem;
}

.footer-compact .footer-links a {
  padding: 0.25rem 0.5rem;
  font-size: 0.8rem;
}

.gradient-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.02),
    rgba(0, 187, 255, 0.02)
  );
  pointer-events: none;
}

.footer-content {
  position: relative;
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.footer-info {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.copyright {
  margin: 0;
  font-size: 0.9rem;
  color: rgba(255, 255, 255, 0.7);
}

.footer-links {
  display: flex;
  gap: 1.5rem;
}

.footer-links a {
  color: rgba(255, 255, 255, 0.7);
  text-decoration: none;
  font-size: 0.9rem;
  transition: all 0.3s ease;
  padding: 0.5rem 1rem;
  border-radius: 4px;
}

.footer-links a:hover {
  color: #fff;
  background: linear-gradient(
    135deg,
    rgba(0, 255, 170, 0.1),
    rgba(0, 187, 255, 0.1)
  );
}

.version {
  margin: 0;
  font-size: 0.8rem;
  color: rgba(255, 255, 255, 0.5);
  font-weight: 300;
}

@media (max-width: 768px) {
  .footer-content {
    flex-direction: column;
    gap: 1rem;
    text-align: center;
  }

  .footer-info {
    align-items: center;
    text-align: center;
  }

  .footer-links {
    gap: 1rem;
  }
}
</style>
