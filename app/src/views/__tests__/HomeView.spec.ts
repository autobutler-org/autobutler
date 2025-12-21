import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import HomeView from '@/views/HomeView.vue'
import router from '@/router'

describe('HomeView', () => {
  it('renders properly', async () => {
    const wrapper = mount(HomeView, {
      props: { msg: 'Autobutler' },
      global: {
        plugins: [router],
      },
    })
    await router.isReady?.()
    expect(wrapper.text()).toContain('Autobutler')
  })
})
