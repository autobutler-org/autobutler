import { describe, it, expect } from 'vitest'

import { mount } from '@vue/test-utils'
import HomeView from '@/views/HomeView.vue'

describe('HomeView', () => {
  it('renders properly', () => {
    const wrapper = mount(HomeView, { props: { msg: 'Autobutler' } })
    expect(wrapper.text()).toContain('Autobutler')
  })
})
