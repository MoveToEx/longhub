import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";
import type { Image, Version } from "@/shared/lib/types";

type UserImagesResponse = {
  total: number,
  images: (Pick<Version, 'text' | 'rating'> & Pick<Image, 'id' | 'imageUrl' | 'currentVersionId'>)[]
}

export default function useUserImages(userId: number, offset: number = 0, limit: number = 24) {
  return useTaggedSWR<[number, number, number], UserImagesResponse>({
    id: 'user-images',
    args: [userId, offset, limit],
    fetcher: async (userId, offset, limit) => {
      const response = await api.get(`/user/${userId}/image`, {
        params: {
          offset: offset.toString(),
          limit: limit.toString(),
        },
      });
      return response.data.data;
    },
    tags: ['user', 'image'],
  });
}
