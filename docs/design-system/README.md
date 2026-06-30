# Coffee POS Design System

Status: Draft

Source image: `docs/screen-captures/00.foundation-styling.png`

This design system translates the supplied plant-commerce mobile styling into a practical Coffee POS interface language. The visual direction is soft, light, and green-accented, but the POS product must stay faster and denser than the reference image because cashiers will use it repeatedly during live service.

## Design Intent

- Make the app feel calm at the counter: light mint backgrounds, white surfaces, soft shadows, and deep navy text.
- Preserve speed and clarity: compact controls, obvious totals, clear payment states, and readable labels.
- Use green as the main action color for available, selected, paid, and successful states.
- Use glass effects only on large shells, floating bars, and overlays. Data-dense controls should stay solid for readability.
- Keep production implementation simple: Preact, TypeScript, plain CSS, CSS custom properties, and relative `/api/...` calls.

## Documentation Map

- [Foundation Styling Analysis](./foundation-styling-analysis.md): section-by-section analysis of the supplied image and how it maps to Coffee POS.
- [Design Tokens](./tokens.md): color, typography, spacing, radius, shadow, motion, and responsive tokens.
- [Component Guidelines](./component-guidelines.md): implementation rules for form fields, dropdowns, dialogs, cards, options, toasts, tables, pagination, and date pickers.

## Application Rules

- Start every authenticated POS screen on the gradient app background.
- Put the main workflow inside a light app shell, not a dark dashboard.
- Prefer white or near-white cards for order entry, menu items, cart lines, and reports.
- Use active green pills for selected categories, selected options, and primary checkout actions.
- Avoid one-color green interfaces. Pair mint and green with deep navy text, warm cream, pale blue-gray, and status colors.
- Keep card radius at `8px` for repeated content cards. Reserve larger radii for app shells, modals, bottom sheets, and floating navigation.
- Do not use decorative gradient blobs or oversized marketing sections inside the POS workflow.
- All interactive elements must support keyboard focus, visible focus rings, and touch targets of at least `44px`.

## Implementation Target

When the frontend starts implementing POS workflows, define the tokens from [Design Tokens](./tokens.md) in `frontend/src/styles.css` under `:root`. Component styles should consume semantic variables such as `--color-surface`, `--color-accent`, and `--shadow-card` instead of hard-coded hex values.

## Verification Checklist

- UI works at `320px`, `768px`, `1024px`, and `1440px`.
- Normal body text meets WCAG AA contrast on its background.
- Focus is visible on text fields, buttons, option chips, date cells, and pagination controls.
- Loading, empty, error, disabled, selected, and success states are represented without relying on color alone.
- Long drink names, Rupiah prices, queue numbers, and table values do not overflow their containers.
