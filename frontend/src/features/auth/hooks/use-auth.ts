import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";
import type { Identity, Wrapped } from "@/shared/lib/types";
import { useLocalStorage } from "usehooks-ts";

export default function useAuth() {
  const [session, , reset] = useLocalStorage('nmsl-session', '');

  return {
    ...useTaggedSWR({
      type: '$custom',
      tags: ['user', 'auth'],
      args: [session],
      async fetcher(session) {
        try {
          const response = await api.get<Wrapped<Identity>>('/auth', {
            headers: {
              'Authorization': 'Session ' + session
            }
          });
          return response.data.data;
        }
        catch {
          return null;
        }
      },
    }),
    reset,
  };
}