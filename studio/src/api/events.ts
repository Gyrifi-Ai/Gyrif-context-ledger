export type EventStreamState = "connecting" | "open" | "closed";

type EventSourceLike = Pick<EventSource, "readyState" | "close" | "onopen" | "onerror">;

export type EventSubscription = {
  close: () => void;
  reconnect: () => void;
};

type EventSubscriptionOptions = {
  onState: (state: EventStreamState) => void;
  onReconnect: () => void;
  onExhausted: (exhausted: boolean) => void;
  createSource?: () => EventSourceLike;
  maxAttempts?: number;
  random?: () => number;
};

const baseDelay = 1_000;
const maximumDelay = 30_000;

export function eventRetryDelay(attempt: number, random = Math.random): number {
  const exponential = Math.min(maximumDelay, baseDelay * 2 ** Math.max(0, attempt - 1));
  return Math.round(exponential * (0.8 + random() * 0.4));
}

export function subscribeToEvents({
  onState,
  onReconnect,
  onExhausted,
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
