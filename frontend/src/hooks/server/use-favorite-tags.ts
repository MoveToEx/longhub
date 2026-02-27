import type { Tag } from "@/lib/types";
import useTaggedSWR from "@/lib/swr";

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