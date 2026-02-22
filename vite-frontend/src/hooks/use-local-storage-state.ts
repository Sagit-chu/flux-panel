import { useCallback, useState, type SetStateAction } from "react";

const readStoredValue = <T>(key: string, fallbackValue: T): T => {
  try {
    const rawValue = localStorage.getItem(key);

    if (rawValue === null) {
      return fallbackValue;
    }

    return JSON.parse(rawValue) as T;
  } catch {
    return fallbackValue;
  }
};

export const useLocalStorageState = <T>(
  key: string,
  initialValue: T,
): readonly [T, (value: SetStateAction<T>) => void, () => void] => {
  const [value, setValue] = useState<T>(() =>
    readStoredValue(key, initialValue),
  );

  const setPersistedValue = useCallback(
    (nextValue: SetStateAction<T>) => {
      setValue((prevValue) => {
        const resolvedValue =
          typeof nextValue === "function"
            ? (nextValue as (value: T) => T)(prevValue)
            : nextValue;

        try {
          localStorage.setItem(key, JSON.stringify(resolvedValue));
        } catch {}

        return resolvedValue;
      });
    },
    [key],
  );

  const resetPersistedValue = useCallback(() => {
    setValue(initialValue);
    try {
      localStorage.removeItem(key);
    } catch {}
  }, [initialValue, key]);

  return [value, setPersistedValue, resetPersistedValue] as const;
};
