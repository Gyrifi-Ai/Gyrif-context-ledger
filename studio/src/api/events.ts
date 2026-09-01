import type { DomainEvent, EventKind } from "./types";

export type EventStreamState = "connecting" | "open" | "closed";

type EventSourceLike = {
  readyState: number;
  close: () => void;
  onopen: ((event: Event) => void) | null;
  onerror: ((event: Event) => void) | null;
  addEventListener: (type: string, listener: (event: Event) => void) => void;
};

export type EventSubscription = {
  close: () => void;
  reconnect: () => void;
};

type EventSubscriptionOptions = {
  onState: (state: EventStreamState) => void;
  onReconnect: () => void;
  onExhausted: (exhausted: boolean) => void;
  onEvent?: (event: DomainEvent) => void;
  createSource?: () => EventSourceLike;
  maxAttempts?: number;
  random?: () => number;
};

const baseDelay = 1_000;
const maximumDelay = 30_000;
const eventKinds: EventKind[] = [
  "change.accepted",
  "change.withdrawn",
  "ledger.archived",
  "ledger.unarchived",
  "proposal.created",
  "proposal.evaluated",
  "proposal.approved",
  "proposal.cancelled",
  "release.started",
  "release.completed",
  "release.failed",
  "intent.recovery_required",
  "intent.resolved",
];

export function parseDomainEvent(data: string): DomainEvent | undefined {
  try {
    const value = JSON.parse(data) as Partial<DomainEvent>;
    if (!eventKinds.includes(value.kind as EventKind)) return undefined;
    if (typeof value.ledgerId !== "string" || typeof value.subjectId !== "string" || typeof value.at !== "string") return undefined;
    return value as DomainEvent;
  } catch {
    return undefined;
  }
}

export function eventRetryDelay(attempt: number, random = Math.random): number {
  const exponential = Math.min(maximumDelay, baseDelay * 2 ** Math.max(0, attempt - 1));
  return Math.round(exponential * (0.8 + random() * 0.4));
}

export function subscribeToEvents({
  onState,
  onReconnect,
  onExhausted,
  onEvent,
  createSource = () => new EventSource("/events/v1"),
  maxAttempts = 6,
  random = Math.random,
}: EventSubscriptionOptions): EventSubscription {
  let source: EventSourceLike | undefined;
  let timer: number | undefined;
  let attempts = 0;
  let generation = 0;
  let disconnected = false;
  let disposed = false;

  const clearTimer = () => {
    if (timer !== undefined) globalThis.clearTimeout(timer);
    timer = undefined;
  };

  const connect = () => {
    if (disposed) return;
    clearTimer();
    source?.close();
    const currentGeneration = ++generation;
    const nextSource = createSource();
    source = nextSource;
    onState("connecting");

    if (onEvent) {
      const receive = (rawEvent: Event) => {
        if (disposed || currentGeneration !== generation) return;
        const data = (rawEvent as MessageEvent<unknown>).data;
        if (typeof data !== "string") return;
        const event = parseDomainEvent(data);
        if (event) onEvent(event);
      };
      eventKinds.forEach((kind) => nextSource.addEventListener(kind, receive));
    }

    nextSource.onopen = () => {
      if (disposed || currentGeneration !== generation) return;
      attempts = 0;
      onExhausted(false);
      onState("open");
      if (disconnected) {
        disconnected = false;
        onReconnect();
      }
    };

    nextSource.onerror = () => {
      if (disposed || currentGeneration !== generation) return;
      disconnected = true;
      if (nextSource.readyState !== EventSource.CLOSED) {
        onState("connecting");
        return;
      }

      nextSource.close();
      onState("closed");
      attempts += 1;
      if (attempts > maxAttempts) {
        onExhausted(true);
        return;
      }
      timer = globalThis.setTimeout(connect, eventRetryDelay(attempts, random));
    };
  };

  const reconnect = () => {
    if (disposed) return;
    attempts = 0;
    disconnected = true;
    onExhausted(false);
    connect();
  };

  const close = () => {
    if (disposed) return;
    disposed = true;
    generation += 1;
    clearTimer();
    source?.close();
    source = undefined;
  };

  connect();
  return { close, reconnect };
}

type LedgerEventSubscriptionOptions = {
  onState?: (state: EventStreamState) => void;
  onReconnect?: () => void;
  onExhausted?: (exhausted: boolean) => void;
  createSource?: (url: string) => EventSourceLike;
  maxAttempts?: number;
  random?: () => number;
};

export function subscribeToLedgerEvents(
  ledgerId: string,
  handler: (event: DomainEvent) => void,
  {
    onState = () => undefined,
    onReconnect = () => undefined,
    onExhausted = () => undefined,
    createSource = (url) => new EventSource(url),
    maxAttempts,
    random,
  }: LedgerEventSubscriptionOptions = {},
): EventSubscription {
  const url = ledgerId ? `/events/v1?ledgerId=${encodeURIComponent(ledgerId)}` : "/events/v1";
  return subscribeToEvents({
    onState,
    onReconnect,
    onExhausted,
    onEvent: handler,
    createSource: () => createSource(url),
    maxAttempts,
    random,
  });
}
