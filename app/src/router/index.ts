import { createRouter, createWebHistory } from 'vue-router';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: import('@/views/HomeView.vue'),
    },
    {
      path: '/cirrus/:pathMatch(.*)*',
      name: 'cirrus',
      component: import('@/views/CirrusView.vue'),
    },
    {
      path: '/photos',
      name: 'photos',
      component: import('@/views/PhotosView.vue'),
    },
    {
      path: '/books',
      name: 'books',
      component: import('@/views/BooksView.vue'),
    },
    {
      path: '/devices',
      name: 'devices',
      component: import('@/views/DevicesView.vue'),
    },
    {
      path: '/settings',
      name: 'settings',
      component: import('@/views/SettingsView.vue'),
    },
  ],
});

export default router;
