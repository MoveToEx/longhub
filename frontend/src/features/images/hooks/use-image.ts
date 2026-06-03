import type { Image, Tag, UserIdentifier, Version } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type ImageResponse = Image & Pick<Version, 'text' | 'rating' | 'version'> & {
  userIdentifier: UserIdentifier,
  tags: Tag[]
}

export default function useImage(id: number) {
  return useTaggedSWR<[number], ImageResponse>({
    id: 'image',
    args: [id],
    fetcher: async (id) => {
      const response = await api.get(`/image/${id}`);
      return response.data.data;
    },
    tags: ['image'],
  });
}
