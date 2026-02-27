import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './global.css'
import { createHashRouter, RouterProvider } from 'react-router'
import IndexPage from './routes/index.tsx'
import Layout from './layout.tsx'
import ImagesPage from './routes/images.tsx'
import UserPage from './routes/user.tsx'
import FavoritesPage from './routes/favorites.tsx'
import SettingsPage from './routes/settings/index.tsx'
import ImagePage from './routes/image.tsx'

const router = createHashRouter([
  {
    path: '/',
    Component: Layout,
    children: [
      {
        index: true,
        Component: IndexPage,
      },
      {
        path: '/image',
        Component: ImagesPage
      },
      {
        path: '/image/:id',
        Component: ImagePage
      },
      {
        path: '/user/:id',
        Component: UserPage
      },
      {
        path: '/user/settings',
        Component: SettingsPage
      },
      {
        path: '/favorite',
        Component: FavoritesPage
      }
    ]
  }
])

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
)
