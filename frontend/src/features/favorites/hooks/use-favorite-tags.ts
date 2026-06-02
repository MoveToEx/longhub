import type { Tag } from "@/shared/lib/types";
import useTaggedSWR from "@/shared/lib/swr";

type FavoriteTagsResponse = (Tag & {
  count: number,
})[]

export default function useFavoriteTags() {
  return useTaggedSWR<[], FavoriteTagsResponse>({
    type: 'GET',
    url: '/recommend',
    tags: ['tag', 'favorite', 'self'],
    immutable: true
  });
}