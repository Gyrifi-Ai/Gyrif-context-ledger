import { useCallback, useEffect, useRef, useState } from "react";

export type QueryResult<T> = {
  data: T | undefined;
  error: Error | undefined;
  loading: boolean;
  refetching: boolean;
  refetch: () => void;
};

type QueryState<T> = Omit<QueryResult<T>, "refetch">;

function asError(value: unknown): Error {
  return value instanceof Error ? value : new Error("The operation could not be completed.");
}

export function useQuery<T>(key: string, fn: (signal: AbortSignal) => Promise<T>, deps: unknown[]): QueryResult<T> {
  const fnRef = useRef(fn);
  const requestRef = useRef(0);
  const controllerRef = useRef<AbortController | null>(null);
  const mountedRef = useRef(true);
  const [state, setState] = useState<QueryState<T>>({ data: undefined, error: undefined, loading: true, refetching: false });

  fnRef.current = fn;

  const refetch = useCallback(() => {
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    const request = ++requestRef.current;

    setState((current) => ({
      data: current.data,
      error: undefined,
      loading: current.data === undefined,
      refetching: current.data !== undefined,
    }));

    void fnRef.current(controller.signal)
      .then((data) => {
        if (!mountedRef.current || request !== requestRef.current) return;
        setState({ data, error: undefined, loading: false, refetching: false });
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted || !mountedRef.current || request !== requestRef.current) return;
        setState((current) => ({ data: current.data, error: asError(error), loading: false, refetching: false }));
      });
  }, [key, ...deps]);

  useEffect(() => {
    mountedRef.current = true;
    refetch();
    return () => {
      mountedRef.current = false;
      controllerRef.current?.abort();
    };
  }, [refetch]);

  return { ...state, refetch };
}

export type MutationResult<TArgs, TResult> = {
  run: (args: TArgs) => Promise<void>;
  pending: boolean;
  error: Error | undefined;
  result: TResult | undefined;
  reset: () => void;
};

export function useMutation<TArgs, TResult>(fn: (args: TArgs) => Promise<TResult>): MutationResult<TArgs, TResult> {
  const fnRef = useRef(fn);
  const pendingRef = useRef(false);
  const mountedRef = useRef(true);
  const [state, setState] = useState<{ pending: boolean; error: Error | undefined; result: TResult | undefined }>({ pending: false, error: undefined, result: undefined });

  fnRef.current = fn;

  useEffect(() => () => { mountedRef.current = false; }, []);

  const run = useCallback(async (args: TArgs) => {
    if (pendingRef.current) return;
    pendingRef.current = true;
    if (mountedRef.current) setState({ pending: true, error: undefined, result: undefined });
    try {
      const result = await fnRef.current(args);
      if (mountedRef.current) setState({ pending: false, error: undefined, result });
    } catch (error) {
      if (mountedRef.current) setState({ pending: false, error: asError(error), result: undefined });
    } finally {
      pendingRef.current = false;
    }
  }, []);

  const reset = useCallback(() => {
    if (mountedRef.current) setState({ pending: false, error: undefined, result: undefined });
  }, []);

  return { ...state, run, reset };
}