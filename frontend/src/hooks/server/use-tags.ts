import api from "@/lib/axios";
import type { Tag } from "@/lib/types";
import useSWR from "swr";

export type TagResponse = {
  tags: Tag[]
}

export const fetcher = async (prefix: string) => {
  if (prefix.indexOf(' ') !== -1) {
    return { tags: [] } satisfies TagResponse;
  }

  const searchParams = new URLSearchParams({ prefix });

  const response = await api.get('/tag/autocomplete?' + searchParams.toString());

  return response.data.data;
}

export default function useTags(prefix: string) {
  return useSWR<TagResponse>(prefix, fetcher);
}