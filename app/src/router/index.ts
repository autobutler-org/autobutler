import { createRouter, createWebHistory } from 'vue-router'
import CirrusView from '@/views/CirrusView.vue'
import HomeView from '@/views/HomeView.vue'
import BooksView from '@/views/BooksView.vue'
import SettingsView from '@/views/SettingsView.vue'
import PhotosView from '@/views/PhotosView.vue'
import DevicesView from '@/views/DevicesView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
    },
    {
      path: '/cirrus/:pathMatch(.*)*',
      name: 'cirrus',
      component: CirrusView,
    },
    {
      path: '/photos',
      name: 'photos',
      component: PhotosView,
    },
    {
      path: '/books',
      name: 'books',
      component: BooksView,
    },
    {
      path: '/devices',
      name: 'devices',
      component: () => DevicesView,
    },
    {
      path: '/settings',
      name: 'settings',
      component: SettingsView,
    },
  ],
})

export default router
