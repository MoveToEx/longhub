import { createBrowserRouter, Navigate } from 'react-router'
import Layout from '@/app/layout'
import FavoritesPage from '@/features/favorites/pages/favorites-page'
import HomePage from '@/features/home/pages/home-page'
import ImagePage from '@/features/images/pages/image-page'
import ImagesPage from '@/features/images/pages/images-page'
import SettingsPage from '@/features/settings/pages/settings-page'
import UserPage from '@/features/user/pages/user-page'
import SearchPage from '@/features/search/pages/search-page'
import AdminLayout from '@/features/admin/components/admin-layout'
import AdminImagesPage from '@/features/admin/pages/admin-images-page'
import AdminUsersPage from '@/features/admin/pages/admin-users-page'

export const router = createBrowserRouter([
  {
    path: '/',
    Component: Layout,
    children: [
      {
        path: '/admin',
        Component: AdminLayout,
        children: [
          { index: true, element: <Navigate to="/admin/images" replace /> },
          { path: 'images', Component: AdminImagesPage },
          { path: 'users', Component: AdminUsersPage },
        ],
      },
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
