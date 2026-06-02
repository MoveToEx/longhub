import useSWR, { mutate as SWRMutate } from "swr";
import api from "./axios";

type BaseParams = {
  tags: string[],
  disabled?: boolean,
  immutable?: boolean,
}

export type GetParams = BaseParams & {
  type: 'GET',
  url: string,
  query?: Record<string, string>,
};

export type PostParams = BaseParams & {
  type: 'POST',
  url: string,
  query?: Record<string, string>,
  payload?: unknown,
}

export type CustomParams<T extends unknown[], R> = BaseParams & {
  type: '$custom',
  args: T,
  fetcher: (...args: T) => Promise<R>,
}

type Params<T extends unknown[] = unknown[], R = unknown> = GetParams | PostParams | CustomParams<T, R>;

const fetcher = async <T extends unknown[], R>(params: Params<T, R>) => {
  if (params.disabled) {
    return null;
  }
  if (params.type === '$custom') {
    const response = await params.fetcher(...params.args);
    return response;
  }
  else if (params.type === 'GET') {
    const query = new URLSearchParams(params.query);
    const s = query.toString();

    const response = await api.get(params.url + (s.length === 0 ? '' : '?' + s));

    return response.data.data;
  }
  else if (params.type === 'POST') {
    const query = new URLSearchParams(params.query);
    const s = query.toString();

    const response = await api.post(params.url + (s.length === 0 ? '' : '?' + s), params.payload);

    return response.data.data;
  }
}

export default function useTaggedSWR<T extends unknown[], R>(params: Params<T, R>) {
  let options = {};

  if (params.immutable) {
    options = {
      revalidateIfStale: false,
      revalidateOnFocus: false,
      revalidateOnReconnect: false
    }
  }

  return useSWR<R>(params, fetcher, options);
}

type NonNullArray<T> = [T, ...T[]];

export function mutate(...tags: NonNullArray<string>) {
  return SWRMutate((key: Params) => {
    return tags.every(tag => key?.tags?.includes(tag)) && key.immutable
  });
}