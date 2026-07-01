# Ship Dashboard Design System

## 1. Atmosphere & Identity

A compact operations console for a single Mac mini and its private Kubernetes edge. The signature is a calm split between local build inputs and Tailscale-only routing proof: quiet surfaces, crisp status color, and dense information that remains scannable.

## 2. Color

| Role | Token | Light | Dark | Usage |
|------|-------|-------|------|-------|
| Surface/primary | --surface-primary | #f7f8f6 | #101312 | Page background |
| Surface/secondary | --surface-secondary | #ffffff | #171b19 | Panels |
| Surface/elevated | --surface-elevated | #ffffff | #202522 | Forms, logs |
| Text/primary | --text-primary | #121614 | #f2f6f3 | Headings, body |
| Text/secondary | --text-secondary | #53605a | #a8b3ad | Captions |
| Text/tertiary | --text-tertiary | #7b8580 | #727d77 | Disabled, metadata |
| Border/default | --border-default | #dfe5e1 | #2d3531 | Inputs, panels |
| Border/subtle | --border-subtle | #edf1ee | #222925 | Dividers |
| Accent/primary | --accent-primary | #047857 | #34d399 | Primary actions, focus |
| Accent/hover | --accent-hover | #065f46 | #6ee7b7 | Hover |
| Status/success | --status-success | #15803d | #4ade80 | Ready |
| Status/warning | --status-warning | #b45309 | #fbbf24 | Dry-run |
| Status/error | --status-error | #b91c1c | #f87171 | Errors |
| Status/info | --status-info | #2563eb | #60a5fa | Informational |

### Rules
- Accent is reserved for actions, focus, and successful routes.
- Warning is used only for dry-run or pending deployment states.
- Public internet exposure uses the info token and must be labeled in text, not color alone.

## 3. Typography

| Level | Size | Weight | Line Height | Tracking | Usage |
|-------|------|--------|-------------|----------|-------|
| Display | 40px | 700 | 1.15 | 0 | Page title |
| H1 | 32px | 700 | 1.2 | 0 | Section headers |
| H2 | 24px | 650 | 1.3 | 0 | Panel titles |
| H3 | 18px | 650 | 1.35 | 0 | Card titles |
| Body/lg | 17px | 400 | 1.6 | 0 | Lead copy |
| Body | 15px | 400 | 1.55 | 0 | Default text |
| Body/sm | 13px | 400 | 1.45 | 0 | Secondary text |
| Caption | 12px | 600 | 1.35 | 0 | Labels, badges |

### Font Stack
- Primary: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif
- Mono: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace

## 4. Spacing & Layout

Base unit: 4px.

| Token | Value | Usage |
|-------|-------|-------|
| --space-1 | 4px | Tight inline gaps |
| --space-2 | 8px | Compact groups |
| --space-3 | 12px | Input padding |
| --space-4 | 16px | Panel inner padding |
| --space-6 | 24px | Form groups |
| --space-8 | 32px | Page groups |
| --space-12 | 48px | Major sections |

Grid: max width 1180px, two columns above 880px, one column below. Breakpoints: 640px, 880px, 1180px.

## 5. Components

### Panel
- Structure: section with heading, optional badge, content body.
- Variants: default, elevated, log.
- Spacing: --space-4 on mobile, --space-6 on desktop.
- States: default, empty, error.
- Accessibility: headings use real hierarchy.
- Motion: none.

### Field
- Structure: label, input, optional hint.
- Variants: text, checkbox, readonly output.
- Spacing: --space-2 label gap, --space-3 input padding.
- States: default, focus, disabled, error.
- Accessibility: label `for` every input.
- Motion: 120ms border/background transition.

### Button
- Structure: native button with text.
- Variants: primary, secondary.
- Spacing: --space-3 vertical, --space-4 horizontal.
- States: default, hover, active, focus, disabled, loading.
- Accessibility: visible focus outline and disabled state.
- Motion: 120ms transform/background transition.

### Route Badge
- Structure: short label plus host.
- Variants: tailscale-only, internet, dry-run, error.
- Spacing: --space-2 inline.
- States: default only.
- Accessibility: text does not rely on color alone.
- Motion: none.

### Deployment Card
- Structure: service heading, status badges, host metadata, direct URL action, preview viewport, tabbed log body.
- Variants: tailscale-only, internet, dry-run, cluster-managed.
- Spacing: --space-4 inner groups, --space-6 card body.
- States: loading, empty, error, active exposure update, preview unavailable.
- Accessibility: cards use articles, tabs are keyboard accessible, long logs stay selectable.
- Motion: 120ms border/background transition.

### Log Tabs
- Structure: shadcn tabs switching between overview, network requests, and terminal logs.
- Variants: default, empty.
- Spacing: --space-3 between tab trigger and panel.
- States: default, active, focus.
- Accessibility: visible focus rings, preserved text contrast, no color-only status.
- Motion: 120ms background transition only.

### Exposure Switch
- Structure: two native buttons grouped as a segmented control.
- Variants: tailscale, internet.
- Spacing: --space-1 inner gap, --space-2 button padding.
- States: default, active, focus, disabled, loading.
- Accessibility: buttons include `aria-pressed` and visible focus rings.
- Motion: 120ms background transition only.

## 6. Motion & Interaction

| Type | Duration | Easing | Usage |
|------|----------|--------|-------|
| Micro | 120ms | ease-out | Button, input focus |
| Standard | 220ms | ease-in-out | Result panel reveal |

Rules: animate only transform and opacity. Respect reduced motion by disabling non-essential transitions.

## 7. Depth & Surface

Strategy: borders with small tonal shifts. Panels use 1px borders and subtle background differences; no large decorative shadows.
