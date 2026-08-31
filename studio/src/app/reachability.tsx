import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { ApiError, api, subscribeToRequestHealth } from "../api/client";
import { subscribeToLedgerEvents, type EventStreamState, type EventSubscription } from "../api/events";
import type { SystemStatus } from "../api/types";

export const runtimeUnavailableMessage = "Cannot reach the Gyrifi runtime. Displayed data may be out of date.";

export type RuntimeHealth = {
  state: "checking" | "connected" | "degraded" | "offline";
  version?: string;
  commit?: string;
  buildDate?: string;
  inference?: string;
  dependencies?: SystemStatus["health"];
};

type Reachability = {
  health: RuntimeHealth;
  unreachable: boolean;
  retry: () => void;
  streamState: EventStreamState;
  streamExhausted: boolean;
  reconnectStream: () => void;
  registerInvalidation: (ledgerId: string | undefined, listener: () => void) => () => void;
};

const Context = createContext<Reachability | null>(null);
const connectedPollDelay = 30_000;

export function reachabilityRetryDelay(attempt: number): number {
  return Math.min(30_000, 1_000 * 2 ** Math.max(0, attempt - 1));
}

export function ReachabilityProvider({ children }: { children: ReactNode }) {
  const [health, setHealth] = useState<RuntimeHealth>({ state: "checking" });
  const [streamState, setStreamState] = useState<EventStreamState>("connecting");
  const [streamExhausted, setStreamExhausted] = useState(false);
  const attemptsRef = useRef(0);
  const timerRef = useRef<number | undefined>(undefined);
  const controllerRef = useRef<AbortController | undefined>(undefined);
  const probingRef = useRef(false);
  const probeRef = useRef<() => void>(() => undefined);
  const streamRef = useRef<EventSubscription | undefined>(undefined);
  const invalidationsRef = useRef(new Set<{ ledgerId: string | undefined; listener: () => void }>());

  const clearTimer = useCallback(() => {
    if (timerRef.current !== undefined) window.clearTimeout(timerRef.current);
    timerRef.current = undefined;
  }, []);

  const schedule = useCallback((delay: number) => {
    clearTimer();
    if (document.hidden) return;
    timerRef.current = window.setTimeout(() => probeRef.current(), delay);
  }, [clearTimer]);

  const probe = useCallback(async () => {
    if (document.hidden) return;
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    probingRef.current = true;
    const started = performance.now();
    try {
      const status: SystemStatus = await api.status({ signal: controller.signal });
      attemptsRef.current = 0;
      setHealth({
        state: performance.now() - started > 2_000 ? "degraded" : "connected",
        version: status.version,
        commit: status.commit,
        buildDate: status.buildDate,
        inference: status.inference,
        dependencies: status.health,
      });
      schedule(connectedPollDelay);
    } catch (error) {
      if (controller.signal.aborted) return;
      if (error instanceof ApiError && error.kind === "http") {
        attemptsRef.current = 0;
        setHealth((current) => ({ ...current, state: "degraded" }));
        schedule(connectedPollDelay);
        return;
      }
      if (error instanceof ApiError && error.kind === "transport") {
        attemptsRef.current += 1;
        setHealth((current) => ({ ...current, state: "offline" }));
        schedule(reachabilityRetryDelay(attemptsRef.current));
        return;
      }
      attemptsRef.current += 1;
      setHealth((current) => ({ ...current, state: "offline" }));
      schedule(reachabilityRetryDelay(attemptsRef.current));
    } finally {
      if (controllerRef.current === controller) probingRef.current = false;
    }
  }, [schedule]);

  probeRef.current = () => { void probe(); };

  const retry = useCallback(() => {
    attemptsRef.current = 0;
    clearTimer();
    probeRef.current();
  }, [clearTimer]);

  const registerInvalidation = useCallback((ledgerId: string | undefined, listener: () => void) => {
    const registration = { ledgerId, listener };
    invalidationsRef.current.add(registration);
    return () => invalidationsRef.current.delete(registration);
  }, []);

  const reconnectStream = useCallback(() => streamRef.current?.reconnect(), []);

  useEffect(() => {
    const unsubscribe = subscribeToRequestHealth((requestHealth) => {
      if (requestHealth.reachable) {
        attemptsRef.current = 0;
        setHealth((current) => current.state === "offline" || current.state === "checking" ? { ...current, state: "connected" } : current);
        schedule(connectedPollDelay);
        return;
      }
      setHealth((current) => ({ ...current, state: "offline" }));
      if (!probingRef.current && attemptsRef.current === 0) {
        attemptsRef.current = 1;
        schedule(reachabilityRetryDelay(attemptsRef.current));
      }
    });
    probeRef.current();

    const visibilityChange = () => {
      if (document.hidden) {
        controllerRef.current?.abort();
        clearTimer();
      } else {
        retry();
      }
    };
    document.addEventListener("visibilitychange", visibilityChange);
    return () => {
      unsubscribe();
      controllerRef.current?.abort();
      clearTimer();
      document.removeEventListener("visibilitychange", visibilityChange);
    };
  }, [clearTimer, retry, schedule]);

  useEffect(() => {
    const subscription = subscribeToLedgerEvents("", (event) => {
      invalidationsRef.current.forEach((registration) => {
        if (!registration.ledgerId || registration.ledgerId === event.ledgerId) registration.listener();
      });
    }, {
      onState: setStreamState,
      onExhausted: setStreamExhausted,
      onReconnect: () => invalidationsRef.current.forEach((registration) => registration.listener()),
    });
    streamRef.current = subscription;
    return () => {
      subscription.close();
      streamRef.current = undefined;
    };
  }, []);

  const value = useMemo<Reachability>(() => ({
    health,
    unreachable: health.state === "offline",
    retry,
    streamState,
    streamExhausted,
    reconnectStream,
    registerInvalidation,
  }), [health, reconnectStream, registerInvalidation, retry, streamExhausted, streamState]);

  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useReachability(): Reachability {
  const value = useContext(Context);
  if (!value) throw new Error("Reachability provider is missing");
  return value;
}

export function useSystemStatus(): RuntimeHealth {
  return useReachability().health;
}