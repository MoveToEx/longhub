import type { AppKey } from "@/lib/types";
import useTaggedSWR from "@/lib/swr";

type AppKeyResponse = AppKey[];

export default function useAppKeys() {
  return useTaggedSWR<[], AppKeyResponse>({
    type: 'GET',
    url: '/user/appkey',
    tags: ['appkey', 'self']
  });
}