import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'ktayl Policy Service',
  tagline: 'Insurance policy lifecycle microservice — CERT-1',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://andrelair-platform.github.io',
  baseUrl: '/ktayl-policy-service/',
  organizationName: 'andrelair-platform',
  projectName: 'ktayl-policy-service',
  trailingSlash: false,

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  markdown: {
    mermaid: true,
  },
  themes: ['@docusaurus/theme-mermaid'],

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: '/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    navbar: {
      title: 'ktayl Policy Service',
      items: [
        {
          href: 'https://andrelair-platform.github.io/minicloud-platform-docs/',
          label: 'Platform Docs',
          position: 'right',
        },
        {
          href: 'https://github.com/andrelair-platform/ktayl-policy-service',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          label: 'Platform Docs',
          href: 'https://andrelair-platform.github.io/minicloud-platform-docs/',
        },
        {
          label: 'GitHub',
          href: 'https://github.com/andrelair-platform/ktayl-policy-service',
        },
      ],
      copyright: `ktayl-solution IS — andrelair-platform`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['go', 'sql', 'bash'],
    },
    mermaid: {
      theme: {light: 'neutral', dark: 'dark'},
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
