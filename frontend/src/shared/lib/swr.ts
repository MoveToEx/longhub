import useSWR, { mutate as SWRMutate, type SWRConfiguration } from "swr";

export type Params<T extends unknown[] = [], R = unknown> = {
  id: string;
  args: T;
  fetcher: (...args: T) => Promise<R>;
  tags: string[];
  immutable?: boolean;
  config?: SWRConfiguration;
};

function exclude<T>(obj: T, key: keyof T) {
  const { [key]: _, ...result } = obj;
  return result;
}

export default function useTaggedSWR<Args extends unknown[], Result>(
  params: Params<Args, Result>,
) {
  return useSWR<Result>(
    exclude(params, "fetcher"),
    async ({ args }: Params<Args, Result>) => {
      const result = await params.fetcher(...args);
      return result;
    },
    params.config,
  );
}

export function mutate(...tags: [string, ...string[]]) {
  return SWRMutate((key: Params) => {
    return tags.every((tag) => key?.tags?.includes(tag)) && !key.immutable;
  });
}
