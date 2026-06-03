import type { UserIdentifier } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type UserResponse = UserIdentifier & {
  versions: number,
  images: number,
};

export default function useUser(id: number) {
  return useTaggedSWR<[number], UserResponse>({
    id: 'user',
    args: [id],
    fetcher: async (id) => {
      const response = await api.get(`/user/${id}`);
      return response.data.data;
    },
    tags: ['user'],
  });
}
