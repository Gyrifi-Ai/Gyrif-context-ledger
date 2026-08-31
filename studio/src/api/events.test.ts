import { afterEach, describe, expect, it, vi } from "vitest";
import { eventRetryDelay, parseDomainEvent, subscribeToEvents, subscribeToLedgerEvents, type EventStreamState } from "./events";

class FakeEventSource {
  static readonly CLOSED = 2;
  readyState = 0;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  close = vi.fn();
  listeners = new Map<string, Array<(event: Event) => void>>();
  addEventListener = vi.fn((type: string, listener: (event: Event) => void) => {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener]);
  });
  emit(type: string, data: string) {
    this.listeners.get(type)?.forEach((listener) => listener({ data } as MessageEvent<string>));
  }
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

  it("parses and dispatches typed domain events for the requested ledger", () => {
    vi.stubGlobal("EventSource", FakeEventSource);
    const source = new FakeEventSource();
    const handler = vi.fn();
    let requestedURL = "";
    const subscription = subscribeToLedgerEvents("ldg one", handler, {
      createSource: (url) => {
        requestedURL = url;
        return source;
      },
    });
    source.readyState = 1;
    source.onopen?.();
    source.emit("change.accepted", JSON.stringify({ kind: "change.accepted", ledgerId: "ldg one", subjectId: "chg_one", at: "2026-08-31T07:00:00Z" }));
    source.emit("change.accepted", "not-json");

    expect(requestedURL).toBe("/events/v1?ledgerId=ldg%20one");
    expect(handler).toHaveBeenCalledOnce();
    expect(handler).toHaveBeenCalledWith({ kind: "change.accepted", ledgerId: "ldg one", subjectId: "chg_one", at: "2026-08-31T07:00:00Z" });
    subscription.close();
  });

  it("rejects unknown or incomplete domain event payloads", () => {
    expect(parseDomainEvent(JSON.stringify({ kind: "unknown", ledgerId: "ldg", subjectId: "id", at: "now" }))).toBeUndefined();
    expect(parseDomainEvent(JSON.stringify({ kind: "release.completed", ledgerId: "ldg" }))).toBeUndefined();
  });
});