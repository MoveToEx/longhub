import { afterEach, expect, it } from 'vitest';
import api from '@/shared/lib/axios';
import { adminGraphQL } from '@/features/admin/lib/graphql';
import type { AxiosAdapter } from 'axios';

const originalAdapter = api.defaults.adapter;
afterEach(() => {
  api.defaults.adapter = originalAdapter;
});

it('uses the requested session instead of a newer stored session', async () => {
  localStorage.setItem('nmsl-session', JSON.stringify('newer-session'));
  const adapter: AxiosAdapter = async (config) => {
    expect(config.headers.get('Authorization')).toBe('Session requested-session');
    expect(config.url).toBe('/admin/graphql');
    return {
      data: { data: { users: { items: [], total: 0 } } },
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  };
  api.defaults.adapter = adapter;
  const data = await adminGraphQL('query { users { total } }', {}, 'requested-session');
  expect(data).toEqual({ users: { items: [], total: 0 } });
});

it('rejects partial GraphQL failures rather than showing partial admin data', async () => {
  api.defaults.adapter = async (config) => ({
    data: { data: { users: { items: [], total: 0 } }, errors: [{ message: 'Query failed' }] },
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
  });
  await expect(adminGraphQL('query { users { total } }', {}, 'session')).rejects.toThrow(
    'Query failed',
  );
});
