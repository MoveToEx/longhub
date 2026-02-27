import { useState, useEffect, type Dispatch, type SetStateAction } from 'react';

interface UseDebouncedStateResult<T> {
  debouncedState: T;
  setState: Dispatch<SetStateAction<T>>;
  instantState: T;
  isPending: boolean;
}

/**
 * A hook to debounce state updates.
 *
 * @param initialValue The default value of the state
 * @param delay The delay in milliseconds
 * @returns Object { debouncedState, setState, instantState, isPending }
 */
export function useDebouncedState<T>(
  initialValue: T,
  delay: number
): UseDebouncedStateResult<T> {
  const [instantState, setInstantState] = useState<T>(initialValue);
  const [debouncedState, setDebouncedState] = useState<T>(initialValue);

  useEffect(() => {
    // Update the debounced state after the specified delay
    const timer = setTimeout(() => {
      setDebouncedState(instantState);
    }, delay);

    // Cleanup: Clear timer if instantState changes or component unmounts
    return () => {
      clearTimeout(timer);
    };
  }, [instantState, delay]);

  // Derived state: check if the values are currently out of sync
  // Note: This works for primitives. For objects/arrays, this relies on reference equality.
  const isPending = instantState !== debouncedState;

  return {
    debouncedState,
    setState: setInstantState,
    instantState,
    isPending,
  };
}