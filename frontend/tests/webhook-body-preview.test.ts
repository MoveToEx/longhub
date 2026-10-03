import assert from 'node:assert/strict';
import { test } from 'node:test';

import { renderWebhookBodyPreview } from '../src/features/settings/lib/webhook-body-preview.ts';

test('default preview preserves the envelope and includes changes only for update', () => {
  for (const event of ['create', 'update', 'delete']) {
    const body = JSON.parse(renderWebhookBodyPreview('', event));
    assert.equal(body.event, event);
    assert.equal(body.data.imageID, 42);
    assert.equal('changes' in body.data, event === 'update');
    assert.deepEqual(
      JSON.parse(renderWebhookBodyPreview('{"event": {{json .Event}}, "data": {{json .Data}}, "initiator": {{json .Initiator}}}', event)),
      body,
    );
  }
});

test('owner and initiator identities are distinct, with Go fields and JSON keys', () => {
  for (const event of ['create', 'update', 'delete']) {
    const expectedOwner = { id: 7, username: 'image-owner' };
    const expectedInitiator = event === 'create' ? expectedOwner : { id: 9, username: 'image-editor' };
    const defaultBody = JSON.parse(renderWebhookBodyPreview('', event));
    assert.deepEqual(defaultBody.data.uploader, expectedOwner);
    assert.deepEqual(defaultBody.initiator, expectedInitiator);
    for (const template of [
      '{"uploader": {{json .Data.Uploader}}, "initiator": {{json .Initiator}}}',
      '{"uploader": {{json .Uploader}}, "initiator": {{json .Initiator}}}',
      '{"uploader": {"id": {{.Data.Uploader.ID}}, "username": {{json .Data.Uploader.Username}}}, "initiator": {"id": {{.Initiator.ID}}, "username": {{json .Initiator.Username}}}}',
    ]) {
      assert.deepEqual(JSON.parse(renderWebhookBodyPreview(template, event)), {
        uploader: expectedOwner,
        initiator: expectedInitiator,
      });
    }
  }
});

test('Go field casing is available while JSON serialization keeps API keys', () => {
  const body = JSON.parse(renderWebhookBodyPreview(
    '{"id": {{.ImageID}}, "text": {{json .Data.Text}}, "tags": {{json .Tags}}, "createdAt": {{json .CreatedAt}}, "changes": {{json .Changes}}, "old": {{json .Data.Changes.text.Old}}, "new": {{json .Changes.text.New}}}',
    'update',
  ));
  assert.equal(body.id, 42);
  assert.equal(body.text, body.new);
  assert.notEqual(body.old, body.new);
  assert.equal(body.createdAt, '2026-01-01T12:00:00Z');
  assert.deepEqual(body.changes.text, { old: body.old, new: body.new });
  assert.deepEqual(body.changes.tags.new, [...body.tags].sort());
});

test('JSON encoding escapes quotes and control characters', () => {
  const body = JSON.parse(renderWebhookBodyPreview(
    '{"text": {{json "quotes: \\"hello\\"\\nbackslash: \\\\ tab: \\t <>&"}}}',
    'create',
  ));
  assert.equal(body.text, 'quotes: "hello"\nbackslash: \\ tab: \t <>&');
});

test('pipelines, conditionals, variables, ranges and root fields are simulated', () => {
  const body = JSON.parse(renderWebhookBodyPreview(
    '{"event": {{.Event | printf "on-%s" | json}}, "updated": {{if eq .Event "update"}}true{{else}}false{{end}}, "tags": [{{range $index, $tag := .Tags}}{{if $index}},{{end}}{{json $tag}}{{end}}], "root": [{{range .Tags}}{{if eq . "featured"}}{{json $.Event}}{{end}}{{end}}]}',
    'update',
  ));
  assert.deepEqual(body, { event: 'on-update', updated: true, tags: ['landscape', 'featured'], root: ['update'] });
});

test('with blocks handle optional update changes', () => {
  const template = '{"previousText": {{with .Changes}}{{with .text}}{{json .Old}}{{else}}null{{end}}{{else}}null{{end}}}';
  assert.equal(JSON.parse(renderWebhookBodyPreview(template, 'create')).previousText, null);
  assert.equal(JSON.parse(renderWebhookBodyPreview(template, 'update')).previousText, 'Original image caption');
});

test('comments and trim markers are interpreted as Go template syntax', () => {
  assert.deepEqual(JSON.parse(renderWebhookBodyPreview(
    '{{/* sample */}} {"id": {{- .Data.ImageID -}} }',
    'delete',
  )), { id: 42 });
});

test('invalid syntax, fields, functions and rendered JSON produce errors', () => {
  for (const template of ['{{', '{{if .Event}}', '{{json .Missing}}', '{{json .Data.Missing}}', '{{unknown .Text}}']) {
    assert.throws(() => renderWebhookBodyPreview(template, 'create'), /Invalid body template/);
  }
  assert.throws(() => renderWebhookBodyPreview('{"text": {{.Text}}}', 'create'), /valid JSON/);
  assert.throws(() => renderWebhookBodyPreview('{}', 'unknown'), /Invalid webhook event/);
});

test('source limits use UTF-8 bytes and rendered output is bounded', () => {
  assert.throws(() => renderWebhookBodyPreview('x'.repeat(64 * 1024 + 1), 'create'), /64 KiB/);
  assert.throws(() => renderWebhookBodyPreview('\u754c'.repeat(22 * 1024), 'create'), /64 KiB/);
  assert.throws(() => renderWebhookBodyPreview(
    '{{range .Tags}}{{range $.Tags}}{{range $.Tags}}{{range $.Tags}}{{range $.Tags}}{{range $.Tags}}{{range $.Tags}}{{range $.Tags}}' + 'x'.repeat(1025) + '{{end}}'.repeat(8),
    'create',
  ), /256 KiB/);
});
