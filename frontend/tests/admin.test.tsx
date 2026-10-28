import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router';
import { SWRConfig } from 'swr';
import { AxiosError } from 'axios';
import AdminLayout from '@/features/admin/components/admin-layout';
import AdminImagesPage from '@/features/admin/pages/admin-images-page';
import AdminUsersPage from '@/features/admin/pages/admin-users-page';
import AppSidebar from '@/app/components/sidebar';
import { SidebarProvider } from '@/shared/components/ui/sidebar';
import {
  ADMIN_PERMISSION,
  ROOT_PERMISSION,
  formatPermissions,
  hasAdminPermission,
  hasRootPermission,
} from '@/shared/lib/const';
import type { Identity } from '@/shared/lib/types';

const mocks = vi.hoisted(() => ({
  auth: {
    data: null as Identity | null,
    session: '',
    isLoading: false,
    mutate: vi.fn(),
    reset: vi.fn(),
  },
  post: vi.fn(),
}));
vi.mock('@/features/auth/hooks/use-auth', () => ({ default: () => mocks.auth }));
vi.mock('@/shared/lib/axios', () => ({ default: { post: mocks.post } }));
vi.mock('@/features/auth/components/login-dialog', () => ({ default: () => null }));

const record = {
  id: '7',
  username: 'table-user',
  email: 'table@example.com',
  permission: 3,
  createdAt: '2026-10-23T00:00:00Z',
  activeImageCount: 2,
  versionCount: 4,
};
const image = {
  id: '9',
  imageUrl: '/test-image.png',
  imageKey: 'storage-key',
  uploader: { id: '7', username: 'uploader' },
  createdAt: record.createdAt,
  updatedAt: record.createdAt,
  deletedAt: null,
  currentVersion: { id: '2', version: 1, text: 'image text', rating: 'none', tags: ['featured'] },
};

function signIn(permission = ADMIN_PERMISSION, session = 'admin-session') {
  mocks.auth.data = { id: 1, username: 'signed-in-user', permission } as Identity;
  mocks.auth.session = session;
}

function mount(path = '/admin/users', sidebar = false) {
  const cache = new Map();
  const app = () => (
    <SWRConfig value={{ provider: () => cache, dedupingInterval: 0, revalidateOnFocus: false }}>
      <MemoryRouter initialEntries={[path]}>
        {sidebar && (
          <SidebarProvider>
            <AppSidebar />
          </SidebarProvider>
        )}
        <Routes>
          <Route path="/admin" element={<AdminLayout />}>
            <Route path="images" element={<AdminImagesPage />} />
            <Route path="users" element={<AdminUsersPage />} />
          </Route>
          <Route path="/" element={<div>Home</div>} />
        </Routes>
      </MemoryRouter>
    </SWRConfig>
  );
  const result = render(app());
  return { ...result, update: () => result.rerender(app()) };
}

beforeEach(() => {
  mocks.auth.data = null;
  mocks.auth.session = '';
  mocks.auth.isLoading = false;
  mocks.auth.mutate.mockReset();
  mocks.post.mockReset();
  mocks.post.mockImplementation(async (_url, { query }) => ({
    data: {
      data: query.includes('AdminImages')
        ? { images: { items: [image], total: 60 } }
        : { users: { items: [record], total: 60 } },
    },
  }));
});

describe('admin access', () => {
  it('accepts Admin or Root for admin access', () => {
    expect(hasAdminPermission(128)).toBe(false);
    expect(hasAdminPermission(8)).toBe(true);
    expect(hasAdminPermission(15)).toBe(true);
    expect(hasAdminPermission(ROOT_PERMISSION)).toBe(true);
    expect(hasAdminPermission(undefined)).toBe(false);
    expect(hasRootPermission(ADMIN_PERMISSION)).toBe(false);
    expect(hasRootPermission(ROOT_PERMISSION)).toBe(true);
    expect(hasRootPermission(undefined)).toBe(false);
  });

  it('waits for identity loading before mounting pages', () => {
    signIn();
    mocks.auth.isLoading = true;
    mount('/admin/images', true);
    expect(screen.getByText('Checking admin access…')).toBeTruthy();
    expect(screen.queryByRole('link', { name: 'Admin' })).toBeNull();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it('allows Root users to open the user administration page without A', async () => {
    signIn(ROOT_PERMISSION);
    mount('/admin/users', true);
    expect(screen.getByRole('link', { name: 'Admin' })).toBeTruthy();
    expect(await screen.findByText('table-user')).toBeTruthy();
  });

  it('removes cached rows and the entrance after logout', async () => {
    signIn();
    const view = mount('/admin/users', true);
    await screen.findByText('table-user');
    mocks.auth.data = null;
    mocks.auth.session = '';
    view.update();
    expect(screen.queryByText('table-user')).toBeNull();
    expect(screen.queryByRole('link', { name: 'Admin' })).toBeNull();
    expect(screen.queryByRole('table')).toBeNull();
  });

  it('isolates cached queries between admin accounts', async () => {
    signIn();
    const view = mount();
    await screen.findByText('table-user');
    mocks.post.mockResolvedValue({
      data: {
        data: { users: { items: [{ ...record, username: 'second-account-result' }], total: 1 } },
      },
    });
    signIn(8, 'another-session');
    view.update();
    expect(await screen.findByText('second-account-result')).toBeTruthy();
    expect(screen.queryByText('table-user')).toBeNull();
    expect(mocks.post.mock.lastCall?.[2].headers.Authorization).toBe('Session another-session');
  });

  it('unmounts tables when current account permissions are revoked', async () => {
    signIn();
    const view = mount('/admin/users', true);
    await screen.findByText('table-user');
    signIn(3);
    view.update();
    expect(screen.queryByText('table-user')).toBeNull();
    expect(screen.queryByRole('link', { name: 'Admin' })).toBeNull();
    expect(screen.getByRole('heading', { name: 'Admin access required' })).toBeTruthy();
  });
});

it.each([
  [0, '-----'],
  [1, '----C'],
  [2, '---E-'],
  [3, '---EC'],
  [4, '--D--'],
  [8, '-A---'],
  [15, '-ADEC'],
  [16, 'R----'],
  [24, 'RA---'],
  [31, 'RADEC'],
  [128, '-----'],
])('formats permission %s as %s', (permission, expected) => {
  expect(formatPermissions(permission)).toBe(expected);
});
describe('admin tables', () => {
  it('fetches pagination and sorting on the server and resets pagination for search', async () => {
    signIn();
    mount();
    const user = userEvent.setup();
    await screen.findByText('table-user');
    await user.click(screen.getByRole('button', { name: 'Go to page 2' }));
    await waitFor(() => expect(mocks.post.mock.lastCall?.[1].variables.input.offset).toBe(25));
    await user.type(screen.getByRole('textbox', { name: 'Search users' }), 'new search');
    await waitFor(() =>
      expect(mocks.post.mock.lastCall?.[1].variables.input).toMatchObject({
        search: 'new search',
        offset: 0,
      }),
    );
    await screen.findByText('table-user');
    await user.click(screen.getByRole('button', { name: 'Username' }));
    await waitFor(() =>
      expect(mocks.post.mock.lastCall?.[1].variables.input).toMatchObject({
        sortBy: 'USERNAME',
        direction: 'DESC',
        offset: 0,
      }),
    );
    await user.click(screen.getByRole('button', { name: 'Username' }));
    await waitFor(() =>
      expect(mocks.post.mock.lastCall?.[1].variables.input.direction).toBe('ASC'),
    );
  });

  it('supports column visibility, page sizes, and refresh', async () => {
    signIn();
    mount();
    const user = userEvent.setup();
    await screen.findByText('table-user');
    await user.click(screen.getByRole('button', { name: 'Columns' }));
    await user.click(await screen.findByRole('menuitemcheckbox', { name: 'Email' }));
    expect(screen.queryByRole('columnheader', { name: 'Email' })).toBeNull();
    await user.keyboard('{Escape}');
    await user.click(screen.getByRole('combobox', { name: 'Rows per page' }));
    await user.click(await screen.findByRole('option', { name: '100' }));
    await waitFor(() => expect(mocks.post.mock.lastCall?.[1].variables.input.limit).toBe(100));
    await screen.findByText('table-user');
    const calls = mocks.post.mock.calls.length;
    await user.click(screen.getByRole('button', { name: 'Refresh users' }));
    await waitFor(() => expect(mocks.post.mock.calls.length).toBeGreaterThan(calls));
  });

  it('filters deleted images and handles nullable current versions', async () => {
    signIn();
    mocks.post.mockResolvedValue({
      data: {
        data: {
          images: {
            items: [{ ...image, currentVersion: null, deletedAt: record.createdAt }],
            total: 1,
          },
        },
      },
    });
    mount('/admin/images');
    const user = userEvent.setup();
    await screen.findByText('No current version');
    await user.click(screen.getByRole('combobox', { name: 'Image status' }));
    await user.click(await screen.findByRole('option', { name: 'Deleted' }));
    await waitFor(() =>
      expect(mocks.post.mock.lastCall?.[1].variables.input).toMatchObject({
        status: 'DELETED',
        offset: 0,
      }),
    );
  });

  it('renders an empty state', async () => {
    signIn();
    mocks.post.mockResolvedValue({ data: { data: { users: { items: [], total: 0 } } } });
    mount();
    expect(await screen.findByText('No users match your filters.')).toBeTruthy();
    expect(screen.getByText('0 results')).toBeTruthy();
  });

  it('shows GraphQL errors and recovers through Retry', async () => {
    signIn();
    mocks.post.mockResolvedValueOnce({
      data: { errors: [{ message: 'Unable to load admin data' }] },
    });
    mount();
    const user = userEvent.setup();
    expect((await screen.findByRole('alert')).textContent).toContain('Unable to load admin data');
    await user.click(screen.getByRole('button', { name: 'Retry' }));
    expect(await screen.findByText('table-user')).toBeTruthy();
  });

  it.each([401, 403])('hides stale data and refreshes identity on HTTP %s', async (status) => {
    signIn();
    mount();
    const user = userEvent.setup();
    await screen.findByText('table-user');
    mocks.post.mockRejectedValue(
      new AxiosError('Access denied', undefined, undefined, undefined, { status } as never),
    );
    await user.click(screen.getByRole('button', { name: 'Refresh users' }));
    await screen.findByText('Admin access is no longer available.');
    expect(screen.queryByText('table-user')).toBeNull();
    await waitFor(() => expect(mocks.auth.mutate).toHaveBeenCalled());
    expect(within(screen.getByRole('alert')).queryByRole('button', { name: 'Retry' })).toBeNull();
  });
});
