export function subscribeToEvents(onChange: () => void): () => void {
  const source = new EventSource("/events/v1");
  source.addEventListener("ledger", onChange);
  source.onerror = () => undefined;
  return () => source.close();
}
