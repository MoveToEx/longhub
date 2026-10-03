import { autocompletion, type CompletionContext } from '@codemirror/autocomplete';
import { json } from '@codemirror/lang-json';
import { indentUnit } from '@codemirror/language';
import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { basicSetup } from 'codemirror';
import { useEffect, useRef } from 'react';

const templateFields = [
  'Event', 'Data', 'ImageID', 'Text', 'Rating', 'Tags', 'ImageURL', 'CreatedAt', 'Changes',
  'Data.ImageID', 'Data.Text', 'Data.Rating', 'Data.Tags', 'Data.ImageURL',
  'Data.CreatedAt', 'Data.Changes',
  'Uploader', 'Uploader.ID', 'Uploader.Username',
  'Data.Uploader', 'Data.Uploader.ID', 'Data.Uploader.Username',
  'Initiator', 'Initiator.ID', 'Initiator.Username',
];

function completeTemplate(context: CompletionContext) {
  const before = context.matchBefore(/\.[\w.]*/);
  if (!before) return null;
  const line = context.state.doc.lineAt(context.pos).text.slice(0, context.pos - context.state.doc.lineAt(context.pos).from);
  if (line.lastIndexOf('{{') <= line.lastIndexOf('}}')) return null;
  return {
    from: before.from,
    options: templateFields.map(field => ({ label: `.${field}`, type: 'property' })),
    validFor: /\.[\w.]*/,
  };
}

const editorTheme = EditorView.theme({
  '&': { backgroundColor: 'var(--background)', color: 'var(--foreground)', fontSize: '12px' },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { fontFamily: 'var(--font-mono, monospace)', overflow: 'auto' },
  '.cm-content': { minHeight: '180px', padding: '8px 0', caretColor: 'var(--foreground)' },
  '.cm-gutters': { backgroundColor: 'var(--muted)', color: 'var(--muted-foreground)', border: 'none' },
  '.cm-activeLine, .cm-activeLineGutter': { backgroundColor: 'var(--muted)' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': { backgroundColor: 'var(--accent)' },
  '.cm-cursor': { borderLeftColor: 'var(--foreground)' },
  '.cm-tooltip': { backgroundColor: 'var(--popover)', color: 'var(--popover-foreground)', borderColor: 'var(--border)' },
  '.cm-tooltip-autocomplete ul li[aria-selected]': { backgroundColor: 'var(--accent)', color: 'var(--accent-foreground)' },
});

export default function WebhookBodyEditor({
  value,
  onChange,
  onBlur,
  readOnly = false,
  jsonSyntax = false,
  label,
}: {
  value: string,
  onChange?: (value: string) => void,
  onBlur?: () => void,
  readOnly?: boolean,
  jsonSyntax?: boolean,
  label: string,
}) {
  const parent = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  const syncing = useRef(false);

  useEffect(() => { onChangeRef.current = onChange; }, [onChange]);

  useEffect(() => {
    if (!parent.current) return;
    const editor = new EditorView({
      parent: parent.current,
      state: EditorState.create({
        extensions: [
          basicSetup,
          ...(jsonSyntax ? [json()] : []),
          indentUnit.of('  '),
          editorTheme,
          EditorView.lineWrapping,
          EditorState.readOnly.of(readOnly),
          EditorView.editable.of(!readOnly),
          EditorView.contentAttributes.of({ 'aria-label': label }),
          EditorView.domEventHandlers({ blur: () => { onBlur?.(); } }),
          autocompletion({ override: [completeTemplate] }),
          EditorView.updateListener.of(update => {
            if (update.docChanged && !syncing.current) {
              onChangeRef.current?.(update.state.doc.toString());
            }
          }),
        ],
      }),
    });
    view.current = editor;
    return () => {
      view.current = null;
      editor.destroy();
    };
  }, [label, onBlur, readOnly, jsonSyntax]);

  useEffect(() => {
    const editor = view.current;
    if (!editor || value === editor.state.doc.toString()) return;
    syncing.current = true;
    try {
      editor.dispatch({ changes: { from: 0, to: editor.state.doc.length, insert: value } });
    }
    finally {
      syncing.current = false;
    }
  }, [value, label, onBlur, readOnly, jsonSyntax]);

  return <div ref={parent} className='max-h-72 min-w-0 overflow-auto rounded-md border focus-within:ring-2 focus-within:ring-ring/50' />;
}
