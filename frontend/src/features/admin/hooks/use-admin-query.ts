import { useEffect } from "react";
import { isAxiosError } from "axios";
import useSWR from "swr";
import useAuth from "@/features/auth/hooks/use-auth";
import { hasAdminPermission } from "@/shared/lib/const";
import { adminGraphQL } from "../lib/graphql";
import type { AdminPage, AdminQueryInput, AdminResource } from "../lib/types";

export default function useAdminQuery<T>(
  resource: AdminResource,
  query: string,
  input: AdminQueryInput,
  enabled = true,
) {
  const auth = useAuth();
  const allowed =
    Boolean(auth.session) && !auth.isLoading && hasAdminPermission(auth.data?.permission);
  const result = useSWR(
    allowed && enabled ? ["admin", auth.session, resource, input] : null,
    async ([, session]) => {
      const data = await adminGraphQL<Record<string, AdminPage<T>>>(query, { input }, session);
      return data[resource];
    },
    { shouldRetryOnError: false, keepPreviousData: false },
  );
  const status = isAxiosError(result.error) ? result.error.response?.status : undefined;
  const unauthorized = status === 401 || status === 403;
  const { mutate } = auth;
  useEffect(() => {
    if (unauthorized) void mutate();
  }, [unauthorized, mutate]);

  return {
    ...result,
    data: allowed && enabled && !result.error ? result.data : undefined,
    unauthorized,
  };
}
