import js from '@eslint/js';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsparser from '@typescript-eslint/parser';
import globals from 'globals';

export default [
    // Global ignores
    {
        ignores: [
            'node_modules/**',
            'build/**',
            'dist/**',
            'playwright-report/**',
            'test-results/**',
            '**/*.min.js',
            'internal/server/public/vendor/**',
        ],
    },
    // JavaScript files configuration
    {
        files: ['internal/server/public/scripts/**/*.js'],
        languageOptions: {
            ecmaVersion: 'latest',
            sourceType: 'module',
            globals: {
                ...globals.browser,
                ...globals.es2021,
                htmx: 'readonly',
                toastr: 'readonly',
                Chart: 'readonly',
                ChartDatasourcePrometheusPlugin: 'readonly',
                ace: 'readonly',
                quill: 'readonly',
            },
        },
        rules: {
            ...js.configs.recommended.rules,
            indent: ['error', 4],
            'linebreak-style': ['error', 'unix'],
            quotes: ['error', 'single'],
            semi: ['error', 'always'],
            'no-unused-vars': ['warn'],
            'no-console': 'off',
        },
    },
    // TypeScript files configuration
    {
        files: ['tests/e2e/**/*.ts'],
        languageOptions: {
            ecmaVersion: 'latest',
            sourceType: 'module',
            parser: tsparser,
            parserOptions: {
                project: false,
            },
        },
        plugins: {
            '@typescript-eslint': tseslint,
        },
        rules: {
            ...js.configs.recommended.rules,
            ...tseslint.configs.recommended.rules,
            '@typescript-eslint/no-unused-vars': ['warn'],
            '@typescript-eslint/no-explicit-any': 'warn',
        },
    },
];
