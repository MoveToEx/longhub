import type { Webhook } from "@/shared/lib/types";
import api from "@/shared/lib/axios";
import useTaggedSWR from "@/shared/lib/swr";

type WebhooksResponse = Webhook[];

export default function useWebhooks() {
  return useTaggedSWR<[], WebhooksResponse>({
    id: 'webhooks',
    args: [],
    fetcher: async () => {
      const response = await api.get('/user/webhook');
      return response.data.data;
    },
    tags: ['settings', 'webhook', 'self'],
  });
}
