import i18n from 'i18next';
import httpBackend from 'i18next-http-backend';
import { initReactI18next } from 'react-i18next';
import { resolveLanguage, useSettingsStore } from '@/stores/settingsStore';

const initialLanguage = resolveLanguage(useSettingsStore.getState().language);

i18n
  .use(httpBackend)
  .use(initReactI18next)
  .init({
    lng: initialLanguage,
    fallbackLng: 'vi',
    load: 'currentOnly',
    debug: false,
    interpolation: {
      escapeValue: false,
    },
    backend: {
      loadPath: '/locales/{{lng}}.json',
    },
  });

export default i18n;
