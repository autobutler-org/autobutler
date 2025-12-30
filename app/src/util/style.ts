export interface CssCustomProperty {
  name: string
  value: string
  element: HTMLElement
}

export const getCssCustomProperties = (
  element: HTMLElement = document.documentElement,
): CssCustomProperty[] => {
  if (!element) return []
  const styles = getComputedStyle(element)
  const customProps: CssCustomProperty[] = []
  for (let i = 0; i < styles.length; i++) {
    const prop = styles[i] || ''
    if (prop.startsWith('--')) {
      customProps.push({
        name: prop,
        value: styles.getPropertyValue(prop).trim(),
        element,
      })
    }
  }
  return customProps
}

export const applyCssCustomProperty = (property: CssCustomProperty): void => {
  property.element.style.setProperty(property.name, property.value)
}
