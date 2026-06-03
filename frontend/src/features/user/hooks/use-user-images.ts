import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";
import type { Image, Version } from "@/shared/lib/types";

type UserImagesResponse = {
  total: number,
  images: (Pick<Version, 'text' | 'rating'> & Pick<Image, 'id' | 'imageUrl' | 'currentVersionId'>)[]
}

export default function useUserImages(userId: number) {
  return useTaggedSWR<[number], UserImagesResponse>({
    id: 'user-images',
    args: [userId],
    fetcher: async (userId) => {
      const response = await api.get(`/user/${userId}/image`);
      return response.data.data;
    },
    tags: ['user', 'image'],
  });
}
