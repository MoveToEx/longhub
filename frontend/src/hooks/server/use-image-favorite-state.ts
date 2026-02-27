import type { Favorite } from "@/lib/types";
import useTaggedSWR from "@/lib/swr";
import useAuth from "./use-auth";

type FavoriteResponse = Favorite | null;

export default function useImageFavoriteState(id: number) {
  const { data } = useAuth();
  
  return useTaggedSWR<[], FavoriteResponse>({
    type: 'GET',
    url: `/favorite/${id}`,
    tags: ['image', 'self', 'favorite'],
    disabled: !data,
  });
}