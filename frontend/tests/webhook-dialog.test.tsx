import { Dialog as BaseDialog } from '@base-ui/react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, it, vi } from 'vitest';
import { AddWebhookDialog, EditWebhookDialog } from '@/features/settings/components/webhook-dialog';
import type { Webhook } from '@/shared/lib/types';

const mocks = vi.hoisted(() => ({ post: vi.fn(), patch: vi.fn(), mutate: vi.fn() }));
vi.mock('@/shared/lib/axios', () => ({ default: { post: mocks.post, patch: mocks.patch } }));
vi.mock('@/features/settings/hooks/use-webhooks', () => ({ default: () => ({ mutate: mocks.mutate }) }));
vi.mock('@/features/settings/components/webhook-body-editor', () => ({
  default: ({ value, onChange, label }: { value: string, onChange?: (value: string) => void, label: string }) => (
    <textarea aria-label={label} value={value} readOnly={!onChange} onChange={event => onChange?.(event.target.value)} />
  ),
}));

const webhook: Webhook = {
  id: 1, label: 'Existing hook', endpoint: 'https://example.com/hook', active: true, eventTypes: 7,
  bodyTemplate: '{}', headerTemplates: { Authorization: 'Bearer token', 'X-Event': '{{.Event}}' },
  lastActivatedAt: null, lastResponseStatus: null, failureCount: 0,
};

beforeEach(() => {
  mocks.post.mockReset().mockResolvedValue({});
  mocks.patch.mockReset().mockResolvedValue({});
  mocks.mutate.mockReset().mockResolvedValue(undefined);
});

async function edit() {
  const handle = BaseDialog.createHandle<Webhook>();
  render(<><button onClick={() => handle.openWithPayload(webhook)}>Edit</button><EditWebhookDialog handle={handle} /></>);
  await userEvent.click(screen.getByRole('button', { name: 'Edit', exact: true }));
  await screen.findByRole('dialog', { name: 'Edit webhook' });
}

it('creates a webhook with multiple templated header pairs and previews their values', async () => {
  const handle = BaseDialog.createHandle<void>();
  render(<><button onClick={() => handle.open()}>New</button><AddWebhookDialog handle={handle} /></>);
  await userEvent.click(screen.getByRole('button', { name: 'New' }));
  await userEvent.type(screen.getByLabelText('Label'), 'Custom headers');
  await userEvent.type(screen.getByLabelText('Endpoint URL'), 'https://example.com/hook');
  await userEvent.type(screen.getByLabelText('Signing secret'), 'secret');
  await userEvent.click(screen.getByRole('checkbox', { name: 'Creation' }));
  await userEvent.click(screen.getByRole('button', { name: 'Add header' }));
  fireEvent.change(screen.getByLabelText('Header name'), { target: { value: 'Authorization' } });
  fireEvent.change(screen.getByLabelText('Value template'), { target: { value: 'Bearer token' } });
  await userEvent.click(screen.getByRole('button', { name: 'Add header' }));
  fireEvent.change(screen.getAllByLabelText('Header name')[1], { target: { value: 'X-Image-ID' } });
  fireEvent.change(screen.getAllByLabelText('Value template')[1], { target: { value: '{{.Data.ImageID}}' } });
  await waitFor(() => expect(screen.getByLabelText('Custom headers preview').textContent).toBe('Authorization: Bearer token\nX-Image-ID: 42'));
  fireEvent.change(screen.getAllByLabelText('Value template')[1], { target: { value: '{{.Event}}' } });
  await waitFor(() => expect(screen.getByLabelText('Custom headers preview').textContent).toBe('Authorization: Bearer token\nX-Image-ID: update'));
  await userEvent.click(screen.getByRole('button', { name: 'Add webhook' }));
  await waitFor(() => expect(mocks.post).toHaveBeenCalledWith('/user/webhook', expect.objectContaining({
    headerTemplates: { Authorization: 'Bearer token', 'X-Image-ID': '{{.Event}}' },
  })));
});

it('loads existing headers and saves an empty object when all rows are removed', async () => {
  await edit();
  expect(screen.getAllByLabelText('Header name').map(input => (input as HTMLInputElement).value)).toEqual(['Authorization', 'X-Event']);
  await userEvent.click(screen.getByRole('button', { name: 'Remove header 1' }));
  await userEvent.click(screen.getByRole('button', { name: 'Remove header 1' }));
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }));
  await waitFor(() => expect(mocks.patch).toHaveBeenCalledWith('/user/webhook/1', expect.objectContaining({ headerTemplates: {} })));
});

it('rejects duplicate header names regardless of case before saving', async () => {
  await edit();
  fireEvent.change(screen.getAllByLabelText('Header name')[1], { target: { value: 'authorization' } });
  await userEvent.click(screen.getByRole('button', { name: 'Save changes' }));
  expect(await screen.findByText('Header names must be unique (case-insensitive).')).toBeTruthy();
  expect(mocks.patch).not.toHaveBeenCalled();
});
