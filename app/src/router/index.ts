import CirrusView from '@/views/CirrusView.vue';
import DevicesView from '@/views/DevicesView.vue';
import HomeView from '@/views/HomeView.vue';
import PhotosView from '@/views/PhotosView.vue';
import { createRouter, createWebHistory } from 'vue-router';

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
    // TODO: This is incomplete is why it is hidden
    // {
    //   path: '/books',
    //   name: 'books',
    //   component: BooksView,
    // },
    {
      path: '/devices',
      name: 'devices',
      component: DevicesView,
    },
    // TODO: This is incomplete is why it is hidden
    // {
    //   path: '/settings',
    //   name: 'settings',
    //   component: SettingsView,
    // },
  ],
});

export default router;
