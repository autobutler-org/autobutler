import { createPinia } from 'pinia';
import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
import ConfigService from './services/configService';

const bootstrapAsync = async () => {
  const app = createApp(App);
  app.use(createPinia());
  await ConfigService.initAsync();
  app.use(router);
  app.mount('#app');
};

bootstrapAsync();
