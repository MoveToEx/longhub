import type { useSWRConfig } from "swr";

// Public hooks use tagged objects; search and admin hooks use array keys.
export function invalidateDeletionCaches(mutate: ReturnType<typeof useSWRConfig>["mutate"]) {
  return mutate(
    (key) => {
      if (Array.isArray(key)) {
        return key[0] === "admin" || key[0] === "/image/search" || key[0] === "/image/quick-search";
      }
      return Boolean(
        key &&
        typeof key === "object" &&
        "tags" in key &&
        Array.isArray(key.tags) &&
        !key.tags.includes("auth") &&
        key.tags.some((tag: string) => ["image", "favorite", "user", "deletion"].includes(tag)),
      );
    },
    undefined,
    { revalidate: true },
  );
}
