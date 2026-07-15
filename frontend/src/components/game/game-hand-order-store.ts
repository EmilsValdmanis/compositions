import { useSelector } from "@tanstack/react-store";
import { createStore } from "@tanstack/store";

type HandOrderState = {
  orders: Record<string, string[]>;
};

const STORAGE_KEY = "compositions.game-hand-order:v1";
const EMPTY_ORDER: string[] = [];

function sameStringArray(a: string[], b: string[]) {
  return a.length === b.length && a.every((value, index) => value === b[index]);
}

function readInitialState(): HandOrderState {
  if (typeof window === "undefined") {
    return { orders: {} };
  }

  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      return { orders: {} };
    }

    const parsed = JSON.parse(stored);
    if (!parsed || typeof parsed !== "object" || typeof parsed.orders !== "object") {
      return { orders: {} };
    }

    const orders = Object.fromEntries(
      Object.entries(parsed.orders).filter(
        ([key, value]) => typeof key === "string" && Array.isArray(value),
      ),
    ) as Record<string, string[]>;

    return { orders };
  } catch {
    return { orders: {} };
  }
}

function persistState(state: HandOrderState) {
  if (typeof window === "undefined") {
    return;
  }

  localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}

const handOrderStore = createStore(readInitialState());

export function usePersistedHandOrder(scopeKey: string | null) {
  return useSelector(handOrderStore, (state) =>
    scopeKey ? (state.orders[scopeKey] ?? EMPTY_ORDER) : EMPTY_ORDER,
  );
}

export function setPersistedHandOrder(scopeKey: string | null, order: string[]) {
  if (!scopeKey) {
    return;
  }

  let nextState: HandOrderState | null = null;

  handOrderStore.setState((current) => {
    const currentOrder = current.orders[scopeKey] ?? EMPTY_ORDER;
    if (sameStringArray(currentOrder, order)) {
      return current;
    }

    nextState = {
      ...current,
      orders: {
        ...current.orders,
        [scopeKey]: order,
      },
    };

    return nextState;
  });

  if (nextState) {
    persistState(nextState);
  }
}
