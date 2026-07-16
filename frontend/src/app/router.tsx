import { createBrowserRouter } from 'react-router'
import Layout from '@/app/layout'
import FavoritesPage from '@/features/favorites/pages/favorites-page'
import HomePage from '@/features/home/pages/home-page'
import ImagePage from '@/features/images/pages/image-page'
import ImagesPage from '@/features/images/pages/images-page'
import SettingsPage from '@/features/settings/pages/settings-page'
import UserPage from '@/features/user/pages/user-page'
import SearchPage from '@/features/search/pages/search-page'

export const router = createBrowserRouter([
  {
    path: '/',
    Component: Layout,
    children: [
      {
        index: true,
        Component: HomePage,
      },
      {
        path: '/image',
        Component: ImagesPage,
      },
      {
        path: '/image/:id',
        Component: ImagePage,
      },
      {
        path: '/user/:id',
        Component: UserPage,
      },
      {
        path: '/user/settings',
        Component: SettingsPage,
      },
      {
        path: '/favorite',
        Component: FavoritesPage,
      },
      {
        path: '/search',
        Component: SearchPage,
      },
    ],
  },
])
