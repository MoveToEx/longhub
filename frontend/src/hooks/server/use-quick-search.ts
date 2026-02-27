import useSWR from "swr";
import api from "@/lib/axios";
import type { Image, Version } from "@/lib/types";

type SearchResponse = (Pick<Version, 'text' | 'rating'> & Pick<Image, 'id' | 'imageUrl'> & {
  relevance: number,
  reason: 'favorite' | 'filter'
})[];

export const fetcher = async ([url, keywords]: [string, string[]]) => {
  if (keywords.length === 0 || keywords.length === 1 && keywords[0] === '') {
    return [];
  }

  const response = await api.post(url, {
    keywords
  });

  return response.data.data;
}

export default function useQuickSearch(keywords: string[]) {
  return useSWR<SearchResponse>(['/image/quick-search', keywords], fetcher);
}