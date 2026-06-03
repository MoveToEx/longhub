import type { UserIdentifier, Version } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type VersionResponse = (Version & {
  userIdentifier: UserIdentifier,
  tags: string[]
})[]

export default function useImageVersions(id: number) {
  return useTaggedSWR<[number], VersionResponse>({
    id: 'image-versions',
    args: [id],
    fetcher: async (id) => {
      const response = await api.get(`/image/${id}/version`);
      return response.data.data;
    },
    tags: ['image'],
  });
}
