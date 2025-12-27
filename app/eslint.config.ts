import pluginVitest from '@vitest/eslint-plugin'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import prettierConfig from 'eslint-config-prettier'
import pluginPlaywright from 'eslint-plugin-playwright'
import preferArrowFunctions from 'eslint-plugin-prefer-arrow-functions'
import pluginVue from 'eslint-plugin-vue'
import { globalIgnores } from 'eslint/config'

// To allow more languages other than `ts` in `.vue` files, uncomment the following lines:
// import { configureVueProject } from '@vue/eslint-config-typescript'
// configureVueProject({ scriptLangs: ['ts', 'tsx'] })
// More info at https://github.com/vuejs/eslint-config-typescript/#advanced-setup

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue}'],
  },

  globalIgnores(['**/dist/**', '**/dist-ssr/**', '**/coverage/**']),

  pluginVue.configs['flat/strongly-recommended'],
  vueTsConfigs.recommended,

  {
    ...pluginVitest.configs.recommended,
    files: ['src/**/__tests__/*'],
  },

  {
    ...pluginPlaywright.configs['flat/recommended'],
    files: ['e2e/**/*.{test,spec}.{js,ts,jsx,tsx}'],
  },
  prettierConfig,
  ...(preferArrowFunctions.configs?.all ? [preferArrowFunctions.configs.all] : []),
  // Temporarily disabled rules for Playwright
  {
    files: ['e2e/*.spec.ts'],
    rules: {
      'playwright/no-conditional-in-test': 'off',
      'playwright/no-conditional-expect': 'off',
      'playwright/no-wait-for-timeout': 'off',
      'playwright/no-useless-not': 'off',
      'playwright/prefer-web-first-assertions': 'off',
      'playwright/no-wait-for-selector': 'off',
      'playwright/no-skipped-test': 'off',
    },
  },
  // Enforce no semicolons
  {
    rules: {
      semi: ['error', 'never'],
    },
  },
  // Allow unused variables and arguments that start with _
  {
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
        },
      ],
    },
  },
)
