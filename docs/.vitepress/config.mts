import { defineConfig } from 'vitepress'
import { useSidebar } from 'vitepress-openapi'
import spec from '../spec/openapi.json'

const sidebar = useSidebar({ spec })

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "LONG Hub Docs",
  base: '/',
  head: [['link', { rel: 'icon', href: '/long.jpg' }]],
  description: "Documents for LONG Hub",
  srcExclude: ['README.md', 'node_modules/**', 'scripts/**', '**/*.yaml'],
  sitemap: { hostname: 'https://docs.longhub.top' },
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: 'API', link: '/api/' },
    ],

    sidebar: [
      {
        text: 'General',
        items: [
          { text: 'FAQ', link: '/faq' },
        ]
      },
      {
        text: 'Integration',
        items: [
          {
            text: 'API',
            link: '/api/',
            collapsed: true,
            items: [
              ...sidebar.generateSidebarGroups({
                linkPrefix: '/api/'
              }).map(group => ({
                ...group,
                collapsed: true
              })),
            ]
          },
          {
            text: 'Webhook',
            link: '/webhook'
          },
          {
            text: 'Catalog sync',
            link: '/catalog-sync'
          }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/MoveToEx/longhub' }
    ],
    editLink: {
      pattern: 'https://github.com/MoveToEx/longhub/edit/main/docs/:path'
    },
    search: { provider: 'local' }
  },
  vite: {
    assetsInclude: ['**/*.yaml', '**/*.yml']
  }
})
