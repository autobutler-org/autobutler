import { onUnmounted, ref } from 'vue';

interface UseDragEdgeAutoScrollOptions {
  edgeThreshold?: number;
  maxStepPx?: number;
}

const DEFAULT_EDGE_THRESHOLD = 64;
const DEFAULT_MAX_STEP_PX = 20;

export const useDragEdgeAutoScroll = (
  options: UseDragEdgeAutoScrollOptions = {},
) => {
  const edgeThreshold = options.edgeThreshold ?? DEFAULT_EDGE_THRESHOLD;
  const maxStepPx = options.maxStepPx ?? DEFAULT_MAX_STEP_PX;

  const activeContainer = ref<HTMLElement | null>(null);
  const scrollStep = ref(0);
  let animationFrameId: number | null = null;

  const stopAutoScroll = () => {
    scrollStep.value = 0;
    activeContainer.value = null;
    if (animationFrameId !== null) {
      cancelAnimationFrame(animationFrameId);
      animationFrameId = null;
    }
  };

  const tick = () => {
    const container = activeContainer.value;
    if (!container || scrollStep.value === 0) {
      stopAutoScroll();
      return;
    }

    const previousTop = container.scrollTop;
    container.scrollTop += scrollStep.value;

    if (container.scrollTop === previousTop) {
      stopAutoScroll();
      return;
    }

    animationFrameId = requestAnimationFrame(tick);
  };

  const ensureTicking = () => {
    if (animationFrameId !== null) return;
    animationFrameId = requestAnimationFrame(tick);
  };

  const updateAutoScroll = (event: DragEvent, container: HTMLElement) => {
    if (container.scrollHeight <= container.clientHeight) {
      stopAutoScroll();
      return;
    }

    const bounds = container.getBoundingClientRect();
    const cursorY = event.clientY;

    const distanceToTop = cursorY - bounds.top;
    const distanceToBottom = bounds.bottom - cursorY;

    let nextStep = 0;

    if (distanceToTop >= 0 && distanceToTop < edgeThreshold) {
      const ratio = (edgeThreshold - distanceToTop) / edgeThreshold;
      nextStep = -Math.max(1, Math.round(maxStepPx * ratio));
    } else if (distanceToBottom >= 0 && distanceToBottom < edgeThreshold) {
      const ratio = (edgeThreshold - distanceToBottom) / edgeThreshold;
      nextStep = Math.max(1, Math.round(maxStepPx * ratio));
    }

    if (nextStep === 0) {
      stopAutoScroll();
      return;
    }

    activeContainer.value = container;
    scrollStep.value = nextStep;
    ensureTicking();
  };

  const handleDragOverAutoScroll = (
    event: DragEvent,
    container?: HTMLElement | null,
  ) => {
    const fallbackContainer = event.currentTarget as HTMLElement | null;
    const targetContainer = container ?? fallbackContainer;
    if (!targetContainer) return;

    updateAutoScroll(event, targetContainer);
  };

  onUnmounted(() => {
    stopAutoScroll();
  });

  return {
    handleDragOverAutoScroll,
    stopAutoScroll,
  };
};
