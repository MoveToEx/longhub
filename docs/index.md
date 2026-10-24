---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "LONG Docs"
  text: "Documents for LONG Hub"
  tagline: Where mothercide sets out
  image:
    src: /banner.jpg
    alt: logo
  actions:
    - theme: brand
      text: Main site
      link: https://longhub.top
    - theme: alt
      text: API
      link: /api
    - theme: alt
      text: Catalog sync
      link: /catalog-sync

features:
  - title: Metadata-based Search
    details: Fetch images that suit your needs
  - title: Favorite
    details: Add shortcuts for your favorite images to quickly find them
  - title: Integration
    details: Freely interact with LONG Hub, and even let LONG Hub tell your server what had happened
  - title: Local catalog replication
    details: Download a consistent catalog snapshot and keep it current with webhook notifications
    link: /catalog-sync
---
