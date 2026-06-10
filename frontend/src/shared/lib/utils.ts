import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"
import mime from 'mime';
import {
  PREFERENCE_STORAGE_KEY,
  schema,
} from "@/shared/hooks/use-preference";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}


async function toBlob(element: HTMLImageElement) {
  const canvas = document.createElement('canvas');
  const context = canvas.getContext('2d');
  if (context === null) {
    throw new Error('Unable to get canvas context');
  }

  canvas.width = element.naturalWidth;
  canvas.height = element.naturalHeight;
  context.fillStyle = 'white';
  context.fillRect(0, 0, canvas.width, canvas.height);
  context.drawImage(element, 0, 0);

  const blob: Blob = await new Promise((resolve, reject) => {
    canvas.toBlob(blob => {
      if (blob === null) {
        reject('Cannot convert canvas to blob');
      }
      else {
        resolve(blob);
      }
    }, 'image/png');
  });

  return blob;
}

async function fetchPNG(src: string) {
  const response = await fetch(src, {
    cache: 'no-store'
  });

  if (!response.ok) {
    throw new Error(response.statusText);
  }

  if (response.body === null) {
    throw new Error('Unexpected null body');
  }

  const blob = await response.blob();

  if (blob.type === 'image/png') {
    return blob;
  }

  const url = URL.createObjectURL(blob);

  const image: HTMLImageElement = await new Promise((resolve, reject) => {
    const element = document.createElement('img');

    element.onload = () => resolve(element);
    element.onerror = reject;
    element.src = url;
  });

  URL.revokeObjectURL(url);

  return toBlob(image);
}

async function fetchHtml(src: string) {
  const response = await fetch(src);

  const data = await response.bytes();

  const b64 = data.toBase64();

  const type = mime.getType(src);
  if (!type || !type.startsWith('image')) return '';

  return `<img src="data:${type};base64,${b64}" />`;
}

export function getPreference() {
  const stored = localStorage.getItem(PREFERENCE_STORAGE_KEY);
  if (!stored) return schema.parse({});

  return schema.parse(JSON.parse(stored));
}

export async function copyImage(
  src: string,
  mode = getPreference().copyMode
) {
  const result: Record<string, Promise<string | Blob>> = {};

  if (mode === 'png' || mode === 'combined') {
    result['image/png'] = fetchPNG(src);
  }
  if (mode === 'html' || mode === 'combined') {
    result['text/html'] = fetchHtml(src);
  }

  await navigator.clipboard.write([ new ClipboardItem(result) ]);
}


export function base64ToArray(s: string) {
  const padding = "=".repeat((4 - s.length % 4) % 4)
  const base64 = (s + padding).replace(/-/g, "+").replace(/_/g, "/")

  const raw = atob(base64)
  const arr = new Uint8Array(raw.length)
  for (let i = 0; i < raw.length; i++) {
    arr[i] = raw.charCodeAt(i)
  }
  return arr
}
