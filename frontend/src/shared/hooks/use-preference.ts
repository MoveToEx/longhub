import type { Dispatch, SetStateAction } from "react";
import { useLocalStorage } from "usehooks-ts";
import { z } from "zod";

export const schema = z.object({
  copyMode: z.enum(['html', 'png', 'auto']).default('png'),
});

export type Preference = z.infer<typeof schema>

export const PREFERENCE_STORAGE_KEY = 'longhub-preference';

export default function usePreference() {
  const [preference, setStoredPreference] = useLocalStorage<Preference>(
    PREFERENCE_STORAGE_KEY,
    schema.parse({}),
    {
      deserializer(value) {
        return schema.parse(JSON.parse(value));
      },
    },
  );

  const setPreference: Dispatch<SetStateAction<Preference>> = value => {
    setStoredPreference(current => {
      const next = value instanceof Function ? value(current) : value;
      return schema.parse(next);
    });
  };

  return [
    preference,
    setPreference,
  ] as const;
}
