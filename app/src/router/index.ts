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
    {
      path: '/data-migration',
      name: 'data-migration',
      component: import('@/views/DataMigrationView.vue'),
    },
    {
      path: '/data-migration/google-takeout',
      name: 'google-takeout',
      component: import('@/views/GoogleTakeoutView.vue'),
    },
    {
      path: '/migrationservice',
      name: 'migration-service',
      component: import('@/views/MigrationServiceView.vue'),
    },
  ],
});

export default router;
