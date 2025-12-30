export interface CssCustomProperty {
  name: string
  value: string
  element: HTMLElement
}

export const applyCssCustomProperty = (property: CssCustomProperty): void => {
  property.element.style.setProperty(property.name, property.value)
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

export const propertyNameToKey = (name: string): string =>
  name
    .replace(/^--/, '')
    .split('-')
    .map((part, index) => {
      if (index === 0) return part
      return part.charAt(0).toUpperCase() + part.slice(1)
    })
    .join('')

export const keyToPropertyName = (
  key: string,
  includePrefix: boolean = true,
): string =>
  (includePrefix ? '--' : '') + key.replace(/([A-Z])/g, '-$1').toLowerCase()

export const keyToLabelName = (key: string): string =>
  keyToPropertyName(key, false)
    .split('-')
    .reverse()
    .join('-')
    .replace(/-/g, ' ')
    .replace(/bg/g, 'background')
    .replace(/\b\w/g, (char) => char.toUpperCase())
    .trim()
