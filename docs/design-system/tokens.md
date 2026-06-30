# Design Tokens

These tokens are the implementation baseline for future Coffee POS frontend slices. Names are semantic so individual values can evolve without rewriting components.

## CSS Token Baseline

```css
:root {
  color-scheme: light;

  --font-sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;

  --color-bg-mint: #dcefe4;
  --color-bg-cream: #f7f4e8;
  --color-bg-blue: #e8eef0;
  --color-bg-app: #f5f8f1;

  --color-surface: #fffefa;
  --color-surface-muted: #f3f8f1;
  --color-surface-glass: rgb(255 255 255 / 68%);
  --color-surface-raised: #ffffff;

  --color-text-strong: #11143a;
  --color-text: #26314d;
  --color-text-muted: #6f7a8d;
  --color-text-subtle: #99a4b3;
  --color-text-inverse: #ffffff;

  --color-border: #e2ebe4;
  --color-border-strong: #cbd9d0;

  --color-accent: #4fc386;
  --color-accent-strong: #28a96d;
  --color-accent-soft: #dff6e9;
  --color-accent-pressed: #238d5d;

  --color-success: #2f9e6d;
  --color-success-soft: #e0f6eb;
  --color-warning: #b7791f;
  --color-warning-soft: #fff3d7;
  --color-danger: #c83f4b;
  --color-danger-soft: #fde7ea;
  --color-info: #2b74b8;
  --color-info-soft: #e3f0fb;

  --gradient-app:
    radial-gradient(circle at 8% 10%, rgb(199 233 211 / 74%), transparent 34%),
    radial-gradient(circle at 88% 12%, rgb(222 239 191 / 72%), transparent 32%),
    linear-gradient(105deg, var(--color-bg-mint) 0%, var(--color-bg-blue) 48%, var(--color-bg-cream) 100%);

  --shadow-shell: 0 28px 70px rgb(38 49 77 / 14%);
  --shadow-card: 0 12px 28px rgb(38 49 77 / 10%);
  --shadow-float: 0 16px 36px rgb(40 169 109 / 24%);
  --shadow-focus: 0 0 0 3px rgb(79 195 134 / 28%);

  --radius-card: 8px;
  --radius-control: 18px;
  --radius-shell: 28px;
  --radius-dialog: 24px;
  --radius-pill: 999px;

  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;

  --duration-fast: 120ms;
  --duration-base: 180ms;
  --ease-standard: cubic-bezier(0.2, 0, 0, 1);
}
```

## Color Usage

| Token | Use |
| --- | --- |
| `--gradient-app` | Page background for authenticated POS workflows. |
| `--color-surface` | Primary panels, cards, dialogs, tables, and inputs. |
| `--color-surface-glass` | App shells, floating navigation, and soft overlays only. |
| `--color-text-strong` | Page titles, drink names, queue numbers, totals. |
| `--color-text` | Standard labels and body text. |
| `--color-text-muted` | Helper text, descriptions, inactive labels. |
| `--color-accent` | Primary actions, selected chips, success highlights. |
| `--color-danger` | Errors and destructive actions. |
| `--color-warning` | Payment pending, unprinted paid order, caution states. |
| `--color-info` | Neutral operational notices and reconnecting states. |

## Typography Scale

Use fixed sizes by role. Do not scale font size with viewport width.

| Role | Size | Line Height | Weight | Use |
| --- | ---: | ---: | ---: | --- |
| Display | `32px` | `40px` | `700` | Login title, paid queue number screen. |
| Page title | `24px` | `32px` | `700` | Cashier, orders, reports page headings. |
| Section title | `18px` | `24px` | `700` | Cart, menu category, payment section. |
| Card title | `15px` | `20px` | `700` | Drink names and order row titles. |
| Body | `14px` | `22px` | `500` | Default interface text. |
| Helper | `12px` | `16px` | `500` | Field help, metadata labels, table hints. |
| Button | `13px` | `16px` | `700` | Buttons, chips, segmented controls. |
| Numeric | `16px` | `22px` | `700` | Prices, totals, counts. |

## Spacing

Use the spacing scale from the CSS baseline. Common patterns:

- Field internal padding: `12px 16px`.
- Card padding: `16px`.
- Panel padding: `20px` on mobile, `24px` on tablet and desktop.
- Section gap: `24px`.
- Dense list row gap: `8px` to `12px`.
- Main authenticated shell gap: `16px` mobile, `24px` desktop.

## Radius

- Repeated cards: `--radius-card` (`8px`).
- Text fields and dropdowns: `--radius-control` (`18px`).
- Pills and chips: `--radius-pill`.
- Dialogs and bottom sheets: `--radius-dialog`.
- Page shells and floating navigation: `--radius-shell`.

## Elevation

- Use `--shadow-card` for cards that sit on the app background.
- Use `--shadow-shell` for the main app shell, login panel, and modal surfaces.
- Use `--shadow-float` only for the primary floating action or toast stack.
- Tables and data panels should rely more on borders than shadows.

## Motion

- Hover and focus transitions: `var(--duration-fast)`.
- Dialogs, sheets, dropdowns, and toasts: `var(--duration-base)`.
- Animate opacity and transform only. Avoid layout-shifting animation in cashier workflows.
- Respect `prefers-reduced-motion` by removing transform transitions.

## Breakpoints

| Breakpoint | Width | Target |
| --- | ---: | --- |
| `xs` | `320px` | Small mobile fallback and narrow browser windows. |
| `md` | `768px` | Tablet and small cashier displays. |
| `lg` | `1024px` | Primary cashier laptop/tablet landscape layout. |
| `xl` | `1440px` | Wide cashier terminal or desktop monitor. |
