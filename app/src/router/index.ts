import { createRouter, createWebHistory } from 'vue-router';

import BooksView from '@/views/BooksView.vue';
import CirrusView from '@/views/CirrusView.vue';
import DataMigrationView from '@/views/DataMigrationView.vue';
import DevicesView from '@/views/DevicesView.vue';
import GoogleTakeoutView from '@/views/GoogleTakeoutView.vue';
import HomeView from '@/views/HomeView.vue';
import MigrationServiceView from '@/views/MigrationServiceView.vue';
import PhotosView from '@/views/PhotosView.vue';
import SettingsView from '@/views/SettingsView.vue';

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
      component: DevicesView,
    },
    {
      path: '/settings',
      name: 'settings',
      component: SettingsView,
    },
    {
      path: '/data-migration',
      name: 'data-migration',
      component: DataMigrationView,
    },
    {
      path: '/data-migration/google-takeout',
      name: 'google-takeout',
      component: GoogleTakeoutView,
    },
    {
      path: '/migrationservice',
      name: 'migration-service',
      component: MigrationServiceView,
    },
  ],
});

export default router;
