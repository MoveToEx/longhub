import useSWR from "swr";
import api from "@/shared/lib/axios";
import type { Image, Rating, Version } from "@/shared/lib/types";

export type SearchConditionType =
  | 'tagInclude'
  | 'tagExclude'
  | 'ratingEq'
  | 'textContains'
  | 'uploadedBy';

export type SearchCondition = {
  type: SearchConditionType;
  value: string;
};

export type SearchOrderBy = 'id' | 'uploadDate';
export type SearchOrder = 'asc' | 'desc';

export type SearchRequest = {
  conditions: SearchCondition[];
  orderBy: SearchOrderBy;
  order: SearchOrder;
};

type SearchImage = Pick<Image, 'id' | 'imageUrl'>
  & Pick<Version, 'text'>
  & { rating: Rating; tags: string[] };

type SearchResponse = {
  total: number;
  images: SearchImage[];
};

export default function useSearch(request: SearchRequest, offset: number, limit: number) {
  return useSWR<SearchResponse>(
    ['/image/search', request, offset, limit],
    async ([url]) => {
      const response = await api.post(url, request, { params: { offset, limit } });
      return response.data.data;
    },
  );
}
