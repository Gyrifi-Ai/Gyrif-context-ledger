export function StatusBadge({ value }: { value: string }) {
  const tone = /ready|released|pass|healthy/i.test(value) ? "positive" : /fail|invalid|blocked|error/i.test(value) ? "negative" : "neutral";
  return <span className={`status status--${tone}`}>{value}</span>;
}
