# Studio design system

Status: **target specification.** The current `studio/src/styles.css` is a single flat file of hardcoded hex values with no tokens, no loading states, no focus-visible handling, and no motion. This document defines what it must become. Tickets GRF-201 … GRF-208 implement it.

Read this before writing any Studio markup or CSS.

---

## 1. Design principles

Gyrifi is a governance tool. People use it to make irreversible decisions about production data. The interface must feel like a control room, not a dashboard.

1. **Consequence is visible.** The closer an action is to mutating the target, the heavier its visual weight. `Release` is the only destructive-weight button in the product.
2. **State before chrome.** Every screen answers "what is the current state?" in the first 200 px of vertical space.
3. **Evidence is content, not a tooltip.** Hashes, fingerprints, findings, and before-images are first-class UI, rendered in mono type, always copyable.
4. **No optimistic lies.** The UI never claims something succeeded before the server confirms it. Never shows "connected" without a probe.
5. **Calm by default, loud on exception.** Colour is reserved for status and focus. A healthy screen is near-monochrome.
6. **Density with air.** Governance data is dense. Use a tight type scale and a generous 8 px spacing rhythm rather than shrinking whitespace.
7. **Keyboard first.** Every action reachable by tab, every list navigable, every focus ring visible.

### Anti-patterns — do not ship these

- Decorative illustrations, mascots, or hero gradients.
- Modal dialogs for anything other than a confirmed destructive action.
- Spinners that replace content that is already on screen.
- Toast notifications as the only record of an outcome.
- Hardcoded hex values in a component file.
- Client-side re-derivation of governance rules.

---

## 2. Token system

All tokens live in `studio/src/styles.css` under `:root`. **No component may use a raw colour, size, or duration value.** Dark is the default and only theme for v1; the token names are theme-neutral so a light theme is additive later.

### 2.1 Colour — raw palette

```css
:root {
  /* Neutrals — cool slate ramp */
  --gy-slate-950: #06080c;
  --gy-slate-925: #090c12;
  --gy-slate-900: #0c1017;
  --gy-slate-850: #11161f;
  --gy-slate-800: #161d28;
  --gy-slate-700: #1e2734;
  --gy-slate-600: #2a3546;
  --gy-slate-500: #3c4a5e;
  --gy-slate-400: #5b6b81;
  --gy-slate-300: #8394a9;
  --gy-slate-200: #aebbcb;
  --gy-slate-100: #d6dee8;
  --gy-slate-050: #eef2f7;

  /* Brand — jade. used for identity, focus, and "current". */
  --gy-jade-600: #0f6b52;
  --gy-jade-500: #17916d;
  --gy-jade-400: #2bb98b;
  --gy-jade-300: #5fd9ae;
  --gy-jade-200: #9aeccd;

  /* Semantic accents */
  --gy-amber-400: #d99a2b;
  --gy-amber-300: #f0b855;
  --gy-rose-500:  #b8384c;
  --gy-rose-400:  #e05a6e;
  --gy-rose-300:  #f79aa7;
  --gy-violet-400:#7c6cf0;   /* proposals / review */
  --gy-sky-400:   #3f96d6;   /* informational */
}
```

### 2.2 Colour — semantic aliases

Components use **only** these.

```css
:root {
  /* Surfaces, ascending elevation */
  --surface-base:    var(--gy-slate-950);
  --surface-sunken:  var(--gy-slate-925);
  --surface-raised:  var(--gy-slate-900);
  --surface-overlay: var(--gy-slate-850);
  --surface-inset:   var(--gy-slate-925);   /* inputs, code blocks */

  /* Borders */
  --border-subtle:   var(--gy-slate-800);
  --border-default:  var(--gy-slate-700);
  --border-strong:   var(--gy-slate-600);
  --border-accent:   var(--gy-jade-600);

  /* Text */
  --text-primary:    var(--gy-slate-050);
  --text-secondary:  var(--gy-slate-200);
  --text-tertiary:   var(--gy-slate-300);
  --text-muted:      var(--gy-slate-400);
  --text-inverted:   var(--gy-slate-950);
  --text-accent:     var(--gy-jade-300);

  /* Interactive */
  --action-primary-bg:       var(--gy-jade-500);
  --action-primary-bg-hover: var(--gy-jade-400);
  --action-primary-fg:       var(--gy-slate-950);
  --action-secondary-bg:     var(--gy-slate-800);
  --action-secondary-bg-hover: var(--gy-slate-700);
  --action-secondary-fg:     var(--text-primary);
  --action-danger-bg:        var(--gy-rose-500);
  --action-danger-bg-hover:  var(--gy-rose-400);
  --action-danger-fg:        var(--gy-slate-050);

  /* Status — used by badges, dots, rails */
  --status-neutral-fg: var(--gy-slate-200);  --status-neutral-bg: var(--gy-slate-800);
  --status-info-fg:    var(--gy-sky-400);    --status-info-bg:    #10202e;
  --status-review-fg:  #a99bff;              --status-review-bg:  #1c1836;
  --status-success-fg: var(--gy-jade-300);   --status-success-bg: #0d2a22;
  --status-warning-fg: var(--gy-amber-300);  --status-warning-bg: #2c2210;
  --status-danger-fg:  var(--gy-rose-300);   --status-danger-bg:  #2e1218;

  /* Focus */
  --focus-ring: 0 0 0 2px var(--surface-base), 0 0 0 4px var(--gy-jade-400);
}
```

#### Status colour mapping — normative

| Domain value | Token set |
|---|---|
| `ACCEPTED` | info |
| `READY` | neutral |
| `INVALID`, `BLOCKED` | danger |
| `RELEASED` (Change or Proposal) | success |
| `DRAFT` | neutral |
| `REVIEWED` | review |
| `APPROVED` | success |
| Intent `READY`, `APPLYING`, `VERIFYING` | warning |
| Intent `FINALIZED` | success |
| Intent `RECOVERY_REQUIRED` | danger |
| `HEAD` marker | accent (jade) |

`StatusBadge` MUST map by exact value against this table. The current regex-based tone guessing is replaced (GRF-202).

### 2.3 Typography

```css
:root {
  --font-sans: "Inter var", Inter, ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
  --font-mono: ui-monospace, "SF Mono", "JetBrains Mono", "Cascadia Code", Menlo, monospace;

  --text-2xs:  0.6875rem;  /* 11px — badges, meta */
  --text-xs:   0.75rem;    /* 12px — hashes, table meta */
  --text-sm:   0.8125rem;  /* 13px — labels, secondary */
  --text-base: 0.875rem;   /* 14px — body, default */
  --text-md:   1rem;       /* 16px — card titles */
  --text-lg:   1.25rem;    /* 20px — section headings */
  --text-xl:   1.5rem;     /* 24px — page title */
  --text-2xl:  1.875rem;   /* 30px — hero metric */

  --leading-tight:  1.25;
  --leading-normal: 1.5;
  --leading-relaxed:1.65;

  --weight-normal: 400;
  --weight-medium: 500;
  --weight-semibold: 600;
  --weight-bold: 700;

  --tracking-tight: -0.011em;
  --tracking-wide:   0.06em;
  --tracking-caps:   0.09em;
}
```

Rules:

- Body copy is `--text-base` / `--leading-normal` / `--weight-normal`.
- Page titles are `--text-xl` / `--weight-semibold` / `--tracking-tight`.
- The "eyebrow" label is `--text-2xs` / `--weight-semibold` / `--tracking-caps` / uppercase / `--text-muted`. It is **muted, not jade** — the current jade eyebrow competes with real status colour.
- **Every hash, ID, fingerprint, unit key, and JSON value uses `--font-mono`** at `--text-xs`, with `font-variant-numeric: tabular-nums`.
- Never use font weights above 700. The current file uses 850/900 — remove them.

### 2.4 Space, radius, elevation

```css
:root {
  --space-0: 0;
  --space-1: 0.25rem;  /* 4  */
  --space-2: 0.5rem;   /* 8  */
  --space-3: 0.75rem;  /* 12 */
  --space-4: 1rem;     /* 16 */
  --space-5: 1.5rem;   /* 24 */
  --space-6: 2rem;     /* 32 */
  --space-7: 3rem;     /* 48 */
  --space-8: 4rem;     /* 64 */

  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 14px;
  --radius-full: 999px;

  --elev-1: 0 1px 2px rgb(0 0 0 / 0.32);
  --elev-2: 0 4px 12px rgb(0 0 0 / 0.36);
  --elev-3: 0 12px 32px rgb(0 0 0 / 0.44);

  --shell-sidebar: 248px;
  --shell-max:     1440px;
  --shell-gutter:  var(--space-6);
}
```

All spacing is a multiple of 4 px via these tokens. Arbitrary values like `13px`, `9px`, `34px` (all present today) are forbidden.

### 2.5 Motion

```css
:root {
  --ease-out:   cubic-bezier(0.16, 1, 0.3, 1);
  --ease-inout: cubic-bezier(0.65, 0, 0.35, 1);
  --dur-instant: 80ms;
  --dur-fast:   140ms;
  --dur-normal: 220ms;
  --dur-slow:   360ms;
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

Motion budget:

| Interaction | Duration | Property |
|---|---|---|
| Hover / active feedback | `--dur-instant` | `background-color`, `border-color` |
| Focus ring | none | appears immediately |
| Panel or row entry | `--dur-normal` | `opacity`, `translateY(4px → 0)` |
| Route transition | `--dur-fast` | `opacity` only |
| Skeleton shimmer | 1.4 s loop | `background-position` |
| Progress rail (release) | `--dur-slow` | `width` |

Never animate `height`, `top/left`, or `box-shadow` on a list.

---

## 3. Layout

### 3.1 Shell

```text
┌────────────┬──────────────────────────────────────────────────────────┐
│            │  Topbar  ─ ledger switcher · HEAD chip · status · actions │
│  Sidebar   ├──────────────────────────────────────────────────────────┤
│  248px     │                                                          │
│            │  Page region  (max 1440px, centered, 32px gutter)         │
│  brand     │                                                          │
│  nav       │   ┌─ Page header ─────────────────────────────────────┐   │
│  ─────     │   │ eyebrow / title / description / primary action    │   │
│  runtime   │   └──────────────────────────────────────────────────┘   │
│  footer    │   ┌─ Content ────────────────┐ ┌─ Side rail 340px ──┐    │
│            │   │                          │ │ (optional)         │    │
└────────────┴───┴──────────────────────────┴─┴────────────────────┴────┘
```

- Sidebar is fixed width `--shell-sidebar`, `--surface-sunken`, right border `--border-subtle`.
- Topbar is `56px`, sticky, `--surface-raised` with a `--border-subtle` bottom edge and `backdrop-filter: blur(8px)`.
- Main content grid: `minmax(0, 1fr) 340px` with `--space-5` gap. The side rail collapses below the main column at `1180px`.
- Full single column below `900px`; sidebar becomes a horizontal scrollable nav strip.

### 3.2 Navigation

Nav items are the four product areas only: Ledgers, Changes, Proposals, Releases. Each shows:

- a 16 px inline SVG icon (stroke `currentColor`, 1.5 width — define them in `ui/primitives/icon.tsx`),
- the label,
- an optional count pill on the right (`READY` change count, open proposal count).

Active item: `--surface-overlay` background, `--text-primary` label, and a 2 px jade rail on the left edge (`::before`, `--radius-full`). Hover: `--surface-raised`.

Nav is **disabled except Ledgers** when no ledger is selected, with a tooltip "Select a ledger first". This replaces the current behaviour of rendering four empty-state pages.

### 3.3 Topbar

| Slot | Content |
|---|---|
| Left | Ledger switcher — a button showing the ledger name plus `⌄`, opening a popover list with search. Shows "Select ledger" when empty. |
| Centre-left | HEAD chip: `HEAD · rel_1a2b…` in mono, click-to-copy. `No releases yet` when HEAD is empty. |
| Right | Runtime status dot + label, driven by a real `GET /api/v1/system/status` poll every 30 s. Three states: `Connected` (jade), `Degraded` (amber, request slow or non-200), `Offline` (rose, request failed). Tooltip shows version and inference mode. |

The current hardcoded "Runtime connected" text MUST be removed.

### 3.4 Page header

Every page renders the same header block:

```
EYEBROW LABEL
Page title                                    [ primary action ]
One-sentence description of what this page governs.
```

Title `--text-xl`, description `--text-base` / `--text-muted`, bottom margin `--space-6`.

---

## 4. Component specifications

All components live in `studio/src/ui/`. Props are typed, minimal, and domain-free.

### 4.1 `Button` (`ui/primitives/button.tsx`)

```ts
type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  size?: "sm" | "md";
  loading?: boolean;
  iconLeft?: ReactNode;
};
```

| Variant | Use |
|---|---|
| `primary` | The single most important action on a surface (Create, Approve) |
| `secondary` | Supporting actions (Evaluate, Cancel) |
| `ghost` | Table row actions, icon-only actions |
| `danger` | `Release` and `Rollback` only |

Rules: height `32px` (`sm`) / `36px` (`md`); radius `--radius-sm`; `--weight-medium`; `loading` renders an inline 14 px spinner, sets `aria-busy`, and disables the button without changing its width; `:focus-visible` applies `--focus-ring`; `:disabled` is `opacity: 0.45` with `cursor: not-allowed`.

### 4.2 `StatusBadge` (`ui/patterns/status-badge.tsx`)

```ts
type StatusTone = "neutral" | "info" | "review" | "success" | "warning" | "danger";
type StatusBadgeProps = { label: string; tone: StatusTone; dot?: boolean };
```

Pill, `--radius-full`, `--text-2xs`, `--weight-semibold`, `--tracking-wide`, uppercase, padding `2px 8px`. Optional 6 px leading dot in `currentColor`. **The tone is passed in by the caller**, mapped in a domain-aware helper in `features/` using the table in §2.2. `ui/` must not know what `RECOVERY_REQUIRED` means.

### 4.3 `Panel` (`ui/layout/panel.tsx`) — new

```ts
type PanelProps = {
  title?: ReactNode; eyebrow?: string; description?: ReactNode;
  actions?: ReactNode; footer?: ReactNode; padding?: "none" | "default";
  children: ReactNode;
};
```

`--surface-raised`, `1px solid --border-subtle`, `--radius-lg`, `--elev-1`. Header row: eyebrow + title on the left, `actions` right-aligned, `--space-4` padding, bottom border when a body follows. `padding="none"` for flush tables.

### 4.4 `DataTable` (`ui/patterns/data-table.tsx`) — new

```ts
type Column<T> = {
  key: string; header: ReactNode; width?: string;
  align?: "start" | "end"; mono?: boolean;
  render: (row: T) => ReactNode;
};
type DataTableProps<T> = {
  columns: Column<T>[]; rows: T[]; getRowId: (row: T) => string;
  selectable?: boolean; selectedIds?: string[];
  onSelectionChange?: (ids: string[]) => void;
  onRowClick?: (row: T) => void;
  loading?: boolean; empty?: ReactNode;
};
```

Real `<table>` semantics. Row height `44px`. Header `--text-2xs` uppercase `--text-muted`, sticky. Row hover `--surface-overlay`. Selected row gets a 2 px jade left rail. `loading` renders `--space-1`-rounded skeleton bars, never a replacement spinner. Keyboard: `ArrowUp`/`ArrowDown` move focus, `Space` toggles selection, `Enter` activates `onRowClick`.

### 4.5 `EmptyState` (`ui/feedback/empty-state.tsx`)

```ts
type EmptyStateProps = {
  icon?: ReactNode; title: string; description?: ReactNode;
  action?: ReactNode; variant?: "default" | "compact";
};
```

Centred, `--space-7` vertical padding, 32 px muted icon, title `--text-md --weight-semibold`, description `--text-base --text-muted` capped at `44ch`, optional action button.

### 4.6 `ErrorState` (`ui/feedback/error-state.tsx`) — new

Renders a rose-toned inline block: bold `title`, the server message verbatim in `--font-mono --text-xs`, and a `Retry` button. Used wherever a request can fail. **Every fetch in the app must have a visible error path** — silent `catch` blocks are forbidden.

### 4.7 `Skeleton` (`ui/feedback/skeleton.tsx`) — new

```ts
type SkeletonProps = { width?: string; height?: string; radius?: string; count?: number };
```

`linear-gradient` shimmer over `--surface-overlay`, 1.4 s loop, disabled under `prefers-reduced-motion`.

### 4.8 `Field` (`ui/primitives/field.tsx`) — new

Wraps label + control + hint + error. Label `--text-sm --weight-medium --text-secondary`. Error text `--text-xs --status-danger-fg` with `role="alert"` and `aria-describedby` wiring. Inputs: `--surface-inset`, `1px solid --border-default`, `--radius-sm`, `36px` height, `--space-3` horizontal padding; `:focus-visible` swaps the border to `--border-accent` and applies `--focus-ring`.

### 4.9 `CodeBlock` / `HashChip` (`ui/patterns/`) — new

- `HashChip({ value, label? })` — mono, truncated to `first 10 + …`, click copies the full value and shows a 1.2 s "Copied" state. Used for every ID and hash in the product.
- `CodeBlock({ value, language?, maxHeight? })` — pretty-printed JSON, `--surface-inset`, `--radius-md`, `--text-xs`, scrollable, with a copy button in the top-right.

### 4.10 `Timeline` (`ui/patterns/timeline.tsx`) — new

Vertical rail with nodes. Props: `items` with `{ id, node, title, meta, body, tone, current? }`. `current` renders a jade filled node with a soft glow; the rest are hollow `--border-strong` nodes. Rail is 1 px `--border-default`.

### 4.11 `Stat` (`ui/patterns/stat.tsx`) — new

Label + value + optional delta/tone. Value `--text-2xl --weight-semibold --tracking-tight`, label `--text-2xs` uppercase `--text-muted`. Used in the overview strips.

### 4.12 `ConfirmDialog` (`ui/patterns/confirm-dialog.tsx`) — new

Native `<dialog>` with backdrop `rgb(6 8 12 / 0.72)`. Required for **Release** and **Rollback** only. Must state the concrete consequence, list the affected unit count, and require the primary button to be pressed deliberately (no `autofocus` on the destructive button).

---

## 5. Page designs

### 5.1 Ledgers

```
LEDGERS
Ledgers                                                    [ + New ledger ]
A ledger is a governed namespace with its own inbox, proposals, and release history.

┌ Ledger cards, 2-up responsive grid ───────────────────────────────────┐
│ ┌───────────────────────────────┐ ┌───────────────────────────────┐   │
│ │ ● product-docs      [ACTIVE]  │ │   support-kb                  │   │
│ │ Support knowledge base        │ │   Ticket macros               │   │
│ │ ldg_9f2c…  · 12 ready · 3 rel │ │   ldg_44ab…  · 0 ready · 0 rel│   │
│ └───────────────────────────────┘ └───────────────────────────────┘   │
└───────────────────────────────────────────────────────────────────────┘
```

- Cards, not a form-beside-list split. Creating a ledger opens an inline expanding card in the first grid slot, or a `ConfirmDialog`-style panel — not a permanently occupied side rail.
- Each card shows name, description, `HashChip` for the ID, and counts for ready Changes and Releases.
- The active ledger card gets a jade border and an `ACTIVE` badge.
- Empty state: "No ledgers yet" + description + primary `Create your first ledger`.

### 5.2 Changes

```
DURABLE INBOX
Changes                                        [ Submit change ]  [ Build proposal → ]
Desired-state mutations waiting to be proposed. Nothing here has touched the target.

[ Stat: 12 Ready ] [ Stat: 3 Released ] [ Stat: 0 Invalid ]

┌ Filter bar: [status ▾] [action ▾] [search unit…] ─────────────────────┐
├ DataTable (selectable) ───────────────────────────────────────────────┤
│ ☐ │ SEQ │ UNIT        │ ACTION │ DESIRED FINGERPRINT │ STATUS │ AGE    │
│ ☐ │ 42  │ point/9f21  │ PUT    │ sha256:3a9f…        │ READY  │ 2m ago │
└───────────────────────────────────────────────────────────────────────┘
```

- Selecting rows reveals a sticky bottom action bar: `n selected` + `Create proposal`. This replaces the checkbox list currently buried in a side panel.
- Row click opens a side drawer with the full desired JSON in a `CodeBlock`, both fingerprints, the idempotency key, and timestamps.
- `Submit change` opens a right-hand drawer: `unit`, `action` segmented control, JSON editor with live validation and a `Format` button, and an auto-generated but editable idempotency key.
- JSON validation errors appear inline under the editor, not as a generic red line at the bottom of the form.

### 5.3 Proposals

This is the most important screen in the product and needs a **two-pane review workspace**, not a list of cards with three buttons.

```
CONTEXT PRs
Proposals                                                  [ + New proposal ]
Review batched changes, attach evidence, approve, and release.

┌ List 380px ──────────┐ ┌ Detail ──────────────────────────────────────┐
│ ▸ DRAFT              │ │ August refund policy refresh        [DRAFT]  │
│   August refund…     │ │ pr_7c1e…  ·  hash sha256:81be…  ·  base HEAD │
│   4 changes · 2m     │ │ ────────────────────────────────────────────  │
│ ─────────────────    │ │  ① Changes ② Evidence ③ Approval ④ Release   │
│ ▸ RELEASED           │ │  ●───────────○───────────○──────────○         │
│   Q3 taxonomy        │ │                                               │
│   9 changes · 1d     │ │  ▾ Changes (4)                                │
└──────────────────────┘ │    ordered table with unit / action / fp      │
                         │  ▾ Evidence                                   │
                         │    criteria textarea + Run evaluation         │
                         │    result card: pass/fail, summary, findings  │
                         │  ▾ Approval                                   │
                         │    actor, timestamp, bound hash               │
                         │  ▾ Release                                    │
                         │    [ Release to Qdrant ]  (danger, confirmed) │
                         └───────────────────────────────────────────────┘
```

Required behaviours:

- **A four-step progress rail** across the top of the detail pane: Changes → Evidence → Approval → Release. Completed steps are jade, the current step is outlined, later steps are muted. This is the single clearest expression of the governance model.
- **Gate reasons are explicit.** A disabled `Approve` button must be accompanied by the literal reason: "Run an evaluation first." A disabled `Release` shows "Approval required" or "HEAD moved after this proposal was created — this proposal can no longer be released."
- **Criteria is user input**, persisted per proposal in `localStorage`, with 3–4 starter presets. The current hardcoded criteria string must go.
- **Evidence renders findings** as a list of `{severity, unit, message}` rows with severity tones, plus the model identity and the bound proposal hash.
- **Stale evidence is loud.** If the displayed evidence hash ≠ the proposal hash, show an amber banner: "Evidence was recorded for a different proposal hash and no longer applies."
- Proposal creation is a drawer that reuses the same selectable `DataTable` of `READY` Changes, with explicit ordering controls (the hash is order-sensitive — the UI must show and let the user set the order).

### 5.4 Releases

```
IMMUTABLE HISTORY
Releases
Every release was applied to the target and verified before it was recorded.

┌ Recovery banner (only when intents need attention) ───────── amber ───┐
│ ⚠ 1 release intent requires recovery.            [ Inspect ]          │
└───────────────────────────────────────────────────────────────────────┘

┌ Timeline ─────────────────────────────────────────────────────────────┐
│ ●  rel_4d21…                                        HEAD   2h ago     │
│ │  August refund policy refresh · 4 units · sha256:81be…               │
│ │  [ View plan ]                                                      │
│ ○  rel_9a03…                                               1d ago     │
│ │  Q3 taxonomy · 9 units                                              │
│ │  [ View plan ]  [ Roll back to here ]                               │
└───────────────────────────────────────────────────────────────────────┘
```

- `HEAD` node is jade and filled; all others are hollow.
- `Roll back to here` is `danger` and opens a `ConfirmDialog` that explains: this creates a **new proposal**, does not rewind, and must itself be evaluated, approved, and released. It lists the number of units that would be restored.
- `View plan` opens a drawer showing the operations with unit, action, expected fingerprint, and whether a before-image was retained.
- Recovery banner links to the intent detail (GRF-213).

---

## 6. Interaction states — mandatory coverage

Every data surface implements **five** states. A page is not done until all five exist.

| State | Requirement |
|---|---|
| Loading | Skeleton matching the final layout. Never a centred spinner over an empty page. Never a layout shift on resolve. |
| Empty | `EmptyState` with a title, a one-line explanation of what would appear here, and the action that creates the first item. |
| Error | `ErrorState` with the server's message verbatim and a working `Retry`. |
| Partial / stale | Content stays visible and dims to `opacity: 0.6` during refetch; never unmount data you already have. |
| Populated | The normal case. |

Action buttons additionally implement `idle → loading → success/error`. Success for a mutating action means the affected surface refetches and a 3 s inline confirmation appears near the action — not a floating toast as the only signal.

---

## 7. Accessibility — non-negotiable

- Contrast: body text ≥ 4.5:1, large text and UI borders ≥ 3:1 against their surface. The `--text-muted` on `--surface-raised` pairing has been chosen to clear 4.5:1; do not darken it further.
- `:focus-visible` is required on every interactive element and must use `--focus-ring`. Never `outline: none` without a replacement.
- Semantic HTML: `<nav>`, `<main>`, `<table>`, `<button>`, `<dialog>`. No `div` with `onClick`.
- All icon-only buttons carry `aria-label`.
- Live regions: `role="status"` for evaluation results, `role="alert"` for errors.
- Dialogs trap focus, close on `Escape`, and restore focus to the trigger.
- Full keyboard path for the critical flow: select ledger → select changes → create proposal → evaluate → approve → release.
- `prefers-reduced-motion` disables all non-essential motion.
- Colour is never the sole carrier of meaning — every status badge has a text label.

---

## 8. Implementation constraints

1. **No new runtime dependencies.** Plain CSS with custom properties. No Tailwind, no CSS-in-JS, no component library, no icon package. Icons are hand-written inline SVG in `ui/primitives/icon.tsx`.
2. **Split `styles.css`.** Target structure:
   ```
   studio/src/styles/
   ├── tokens.css        # :root only — every token in §2
   ├── reset.css         # box-sizing, margins, focus defaults
   ├── base.css          # element defaults, typography
   └── components.css    # component classes, BEM-ish: .gy-panel, .gy-panel__header
   ```
   `main.tsx` imports one `styles/index.css` that `@import`s the four.
3. **Class naming:** `gy-{block}__{element}--{modifier}`. Every class is prefixed `gy-` to avoid collisions with embedded content.
4. **No inline `style` attributes** except for genuinely dynamic values (a progress width, a grid `--width` custom property).
5. **`ui/` stays domain-free.** Domain→tone mapping lives in `features/shared/status.ts`.
6. **Every component gets a Vitest test** covering render, the disabled/loading path, and keyboard interaction (GRF-230).
7. `pnpm typecheck` must stay clean under `strict: true`.

---

## 9. Definition of done for any UI ticket

- [ ] Uses only tokens from §2 — no raw hex, px, or ms literals in component CSS.
- [ ] All five interaction states from §6 are implemented.
- [ ] Keyboard-navigable end to end; `:focus-visible` visible on every control.
- [ ] Contrast verified against §7.
- [ ] No governance rule re-derived client-side; gating reasons come from server responses.
- [ ] Errors surfaced verbatim from `error.message`; no swallowed rejections.
- [ ] Responsive at 1440 / 1180 / 900 / 480 px.
- [ ] `pnpm typecheck && pnpm test && pnpm build` all pass.
- [ ] Screenshot or description of before/after recorded in the phase log.
