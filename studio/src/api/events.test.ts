import { afterEach, describe, expect, it, vi } from "vitest";
import { eventRetryDelay, subscribeToEvents, type EventStreamState } from "./events";

class FakeEventSource {
  static readonly CLOSED = 2;
  readyState = 0;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  close = vi.fn();
}

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("event subscription", () => {
  it("reports reconnect state and refetches after reopening", () => {
    vi.useFakeTimers();
    vi.stubGlobal("EventSource", FakeEventSource);
    const sources: FakeEventSource[] = [];
    const states: EventStreamState[] = [];
    const onReconnect = vi.fn();
    const subscription = subscribeToEvents({
      onState: (state) => states.push(state),
      onReconnect,
      onExhausted: vi.fn(),
      createSource: () => {
        const source = new FakeEventSource();
        sources.push(source);
        return source;
      },
      random: () => 0.5,
    });

    sources[0].readyState = 1;
    sources[0].onopen?.();
    sources[0].readyState = FakeEventSource.CLOSED;
    sources[0].onerror?.();
    vi.advanceTimersByTime(1_000);
    sources[1].readyState = 1;
    sources[1].onopen?.();

    expect(states).toEqual(["connecting", "open", "closed", "connecting", "open"]);
    expect(onReconnect).toHaveBeenCalledTimes(1);
    subscription.close();
  });

  it("stops after the retry ceiling and manual reconnect starts a fresh source", () => {
    vi.useFakeTimers();
    vi.stubGlobal("EventSource", FakeEventSource);
    const sources: FakeEventSource[] = [];
    const onExhausted = vi.fn();
    const subscription = subscribeToEvents({
      onState: vi.fn(),
      onReconnect: vi.fn(),
      onExhausted,
      createSource: () => {
        const source = new FakeEventSource();
        sources.push(source);
        return source;
      },
      maxAttempts: 1,
      random: () => 0.5,
    });

    sources[0].readyState = FakeEventSource.CLOSED;
    sources[0].onerror?.();
    vi.advanceTimersByTime(1_000);
    sources[1].readyState = FakeEventSource.CLOSED;
    sources[1].onerror?.();

    expect(onExhausted).toHaveBeenLastCalledWith(true);
    subscription.reconnect();
    expect(sources).toHaveLength(3);
    subscription.close();
  });

  it("closes the source and clears a pending retry on teardown", () => {
    vi.useFakeTimers();
    vi.stubGlobal("EventSource", FakeEventSource);
    const source = new FakeEventSource();
    const createSource = vi.fn(() => source);
    const subscription = subscribeToEvents({ onState: vi.fn(), onReconnect: vi.fn(), onExhausted: vi.fn(), createSource, random: () => 0.5 });
    source.readyState = FakeEventSource.CLOSED;
    source.onerror?.();

    subscription.close();
    vi.runAllTimers();

    expect(source.close).toHaveBeenCalled();
    expect(createSource).toHaveBeenCalledTimes(1);
  });

  it("uses exponential delays with bounded jitter and a thirty-second cap", () => {
    expect(eventRetryDelay(1, () => 0.5)).toBe(1_000);
    expect(eventRetryDelay(3, () => 0.5)).toBe(4_000);
    expect(eventRetryDelay(10, () => 0.5)).toBe(30_000);
  });
});