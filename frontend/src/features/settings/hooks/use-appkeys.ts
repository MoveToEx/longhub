import type { AppKey } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type AppKeyResponse = AppKey[];

export default function useAppKeys() {
  return useTaggedSWR<[], AppKeyResponse>({
    id: 'appkeys',
    args: [],
    fetcher: async () => {
      const response = await api.get('/user/appkey');
      return response.data.data;
    },
    tags: ['appkey', 'self'],
  });
}
