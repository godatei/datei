// @ts-check
import mdx from '@astrojs/mdx';
import sitemap from '@astrojs/sitemap';
import lit from '@semantic-ui/astro-lit';
import icon from 'astro-icon';
import {defineConfig, fontProviders} from 'astro/config';

// https://astro.build/config
export default defineConfig({
  site: 'https://godatei.com',
  integrations: [
    lit(),
    icon({include: {'material-symbols': ['*'], lucide: ['*']}}),
    sitemap(),
    mdx(),
  ],
  fonts: [
    {
      provider: fontProviders.google(),
      name: 'Google Sans',
      cssVariable: '--font-google-sans',
      weights: ['400 700'],
      styles: ['normal', 'italic'],
      display: 'swap',
    },
    {
      provider: fontProviders.google(),
      name: 'Google Sans Code',
      cssVariable: '--font-google-sans-code',
      weights: ['400 600'],
      display: 'swap',
    },
  ],
});
