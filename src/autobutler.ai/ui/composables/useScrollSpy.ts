import { ref, onMounted, onUnmounted, nextTick, readonly } from 'vue'

export const useScrollSpy = (headingSelector = 'h1, h2, h3, h4, h5, h6') => {
  const activeId = ref<string>('')
  const headings = ref<Array<{ id: string; text: string; level: number; element: HTMLElement }>>([])
  
  let observer: IntersectionObserver | null = null
  let isScrolling = false
  let scrollTimeout: NodeJS.Timeout

  const collectHeadings = () => {
    const elements = document.querySelectorAll(headingSelector)
    headings.value = Array.from(elements)
      .filter((el): el is HTMLElement => el instanceof HTMLElement && el.id !== '')
      .map((el) => ({
        id: el.id,
        text: el.textContent || '',
        level: parseInt(el.tagName.substring(1)),
        element: el
      }))
  }

  const createIntersectionObserver = () => {
    if (typeof window === 'undefined') return

    observer = new IntersectionObserver(
      (entries) => {
        if (isScrolling) return

        // Find the entry with the highest intersection ratio that's intersecting
        const intersectingEntries = entries.filter(entry => entry.isIntersecting)
        
        if (intersectingEntries.length > 0) {
          // Sort by intersection ratio and pick the most visible one
          const mostVisible = intersectingEntries
            .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0]
          
          activeId.value = mostVisible.target.id
        } else {
          // If no sections are intersecting, find the closest one above the viewport
          const allHeadings = headings.value
          let closestHeading = null
          let minDistance = Infinity

          for (const heading of allHeadings) {
            const rect = heading.element.getBoundingClientRect()
            const distance = Math.abs(rect.top)
            
            if (rect.top <= 100 && distance < minDistance) {
              minDistance = distance
              closestHeading = heading
            }
          }

          if (closestHeading) {
            activeId.value = closestHeading.id
          }
        }
      },
      {
        rootMargin: '-20% 0px -70% 0px',
        threshold: [0, 0.25, 0.5, 0.75, 1]
      }
    )

    // Observe all headings
    headings.value.forEach(heading => {
      observer?.observe(heading.element)
    })
  }

  const handleScroll = () => {
    isScrolling = true
    clearTimeout(scrollTimeout)
    
    scrollTimeout = setTimeout(() => {
      isScrolling = false
    }, 100)
  }

  const scrollToHeading = (id: string) => {
    const element = document.getElementById(id)
    if (element) {
      // Temporarily disable observer during programmatic scroll
      isScrolling = true
      
      element.scrollIntoView({
        behavior: 'smooth',
        block: 'start'
      })
      
      // Update active immediately for better UX
      activeId.value = id
      
      setTimeout(() => {
        isScrolling = false
      }, 1000) // Allow time for smooth scroll to complete
    }
  }

  const init = async () => {
    await nextTick()
    collectHeadings()
    createIntersectionObserver()
    
    // Set initial active heading
    if (headings.value.length > 0) {
      const firstHeading = headings.value[0]
      const rect = firstHeading.element.getBoundingClientRect()
      if (rect.top <= 200) {
        activeId.value = firstHeading.id
      }
    }
  }

  const cleanup = () => {
    if (observer) {
      observer.disconnect()
      observer = null
    }
    clearTimeout(scrollTimeout)
    window.removeEventListener('scroll', handleScroll)
  }

  onMounted(() => {
    window.addEventListener('scroll', handleScroll, { passive: true })
    init()
  })

  onUnmounted(() => {
    cleanup()
  })

  return {
    activeId: readonly(activeId),
    headings: readonly(headings),
    scrollToHeading,
    refresh: init
  }
} 