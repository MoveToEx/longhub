import { afterAll, afterEach, beforeEach, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { createMemoryRouter, Outlet, RouterProvider } from 'react-router';
import { SWRConfig } from 'swr';
import { router as appRouter } from '@/app/router';
import AppSidebar from '@/app/components/sidebar';
import { SidebarProvider } from '@/shared/components/ui/sidebar';
import type { Identity } from '@/shared/lib/types';

const mocks = vi.hoisted(() => ({
  auth: {
    data: null as Identity | null,
    session: 'admin-session',
    isLoading: false,
    mutate: vi.fn(),
    reset: vi.fn(),
  },
  post: vi.fn(),
}));
vi.mock('@/features/auth/hooks/use-auth', () => ({ default: () => mocks.auth }));
vi.mock('@/shared/lib/axios', () => ({ default: { post: mocks.post } }));
vi.mock('@/features/auth/components/login-dialog', () => ({ default: () => null }));
// Keep the production route tree and sidebar while excluding unrelated app chrome.
vi.mock('@/app/layout', () => ({
  default: () => <SidebarProvider><AppSidebar /><Outlet /></SidebarProvider>,
}));

let router: ReturnType<typeof createMemoryRouter>;

function mount(path: string) {
  router = createMemoryRouter(appRouter.routes, { initialEntries: [path] });
  render(
    <SWRConfig value={{ provider: () => new Map(), revalidateOnFocus: false }}>
      <RouterProvider router={router} />
    </SWRConfig>,
  );
}

beforeEach(() => {
  mocks.auth.data = { id: 1, username: 'admin', permission: 8 } as Identity;
  mocks.post.mockReset();
  mocks.post.mockResolvedValue({ data: { data: { images: { items: [], total: 0 } } } });
});
afterEach(() => router?.dispose());
afterAll(() => appRouter.dispose());

it('reveals the admin entrance and redirects the production admin root to images', async () => {
  mount('/admin');
  expect(screen.getByRole('link', { name: 'Admin' })).toBeTruthy();
  expect(await screen.findByText('No images match your filters.')).toBeTruthy();
  expect(router.state.location.pathname).toBe('/admin/images');
  expect(mocks.post.mock.calls[0][1].variables.input.status).toBe('ACTIVE');
});

it.each(
  ['/admin/images', '/admin/users', '/admin/deletions'].flatMap(path =>
    [null, 3, 128].map(permission => ({ path, permission })),
  ),
)(
  'guards $path for permission $permission before making admin requests',
  ({ path, permission }) => {
    mocks.auth.data = permission === null ? null : { id: 1, username: 'ordinary-user', permission } as Identity;
    mocks.auth.session = permission === null ? '' : 'admin-session';
    mount(path);
    expect(screen.getByRole('heading', { name: 'Admin access required' })).toBeTruthy();
    expect(screen.queryByRole('link', { name: 'Admin' })).toBeNull();
    expect(mocks.post).not.toHaveBeenCalled();
  },
);
