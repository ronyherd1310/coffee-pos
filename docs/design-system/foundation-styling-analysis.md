# Foundation Styling Analysis

Source image: `docs/screen-captures/00.foundation-styling.png`

The reference image shows three mobile screens for a plant shopping app on a soft green gradient canvas. Coffee POS should reuse the visual system, not the subject matter. The result should feel fresh, calm, and tactile while still prioritizing fast cashier operation.

## 1. Overall Canvas

The whole image sits on a low-contrast mint, cream, and blue-gray gradient. The background is not a flat color, and it is not a loud decorative gradient. It creates atmosphere while letting the white app surfaces remain dominant.

Styling cues:

- Background moves from pale mint on the left to pale cream and light green on the right.
- Contrast is intentionally soft, with no hard horizon line.
- Shadows are broad and low opacity, giving the screens a raised paper-like feel.
- The palette is natural but clean: mint, leaf green, white, deep navy, muted gray.

Coffee POS application:

- Use the gradient as the app-level background behind authenticated workflows.
- Keep the gradient subtle enough that printed-ticket previews, order totals, and tables stay readable.
- Avoid decorative shapes, bokeh, or floating orbs. The gradient itself is the background treatment.

## 2. Left Screen: Browse And Selection UI

The left screen is the richest UI section. It includes top navigation, search, category chips, product cards, recently viewed cards, and a floating bottom navigation bar.

Styling cues:

- The screen shell is a translucent pale surface with a large radius.
- Header controls are minimal icon buttons, not text-heavy navigation.
- Search is a pill field with a white translucent fill and small icon affordances.
- Category options are horizontal pills. The active pill is green with white text.
- Product cards are small white surfaces with soft shadows. Images visually rise above the card body.
- Product text uses deep navy for names, muted gray for description, and green for price.
- The bottom navigation is a floating white bar. The selected icon uses green, and the main center action is a raised green circular button.

Coffee POS application:

- Map category pills to drink categories such as Coffee, Non Coffee, Tea, and Snacks.
- Map product cards to menu item cards with drink name, price, and compact metadata.
- Use the search field for menu search or order lookup, but keep it optional in the primary cashier flow.
- Use the floating action pattern sparingly. The POS should not hide critical actions behind a center button; checkout and payment should stay explicit.

## 3. Center Screen: Immersive Featured Object

The center screen is more visual and less operational. It shows a large product image, scan-style corner brackets, a translucent information bar, and circular navigation controls.

Styling cues:

- One large visual object dominates the viewport.
- The shell is tinted mint with heavier blur and reduced content density.
- The bottom information bar is translucent and floats above the content.
- Navigation controls are circular with white translucent fills.
- The scan brackets add a technical overlay while staying thin and white.

Coffee POS application:

- Do not use this layout for routine order entry because it sacrifices speed and density.
- Reuse the treatment for QRIS display, receipt preview, paid-order confirmation, or focused queue-number views.
- The translucent bottom bar pattern can work for "Queue No. 001", payment method, and primary next action after a paid order.
- Keep overlays ornamental only when they do not interfere with order data.

## 4. Right Screen: Detail And Checkout UI

The right screen is a product detail page with a hero image, structured metadata, favorite action, facts row, price, and checkout button.

Styling cues:

- Header has sparse icon controls and generous breathing room.
- Metadata is presented as label-value pairs with muted labels and deep navy values.
- The product detail surface curves upward with a large white panel at the bottom.
- Icon facts use green line icons with small labels.
- Checkout is a green pill button paired with a strong price on the left.
- The favorite button is a green icon, not a full text control.

Coffee POS application:

- Use label-value pairs for payment method, order status, queue number, and cashier-session metadata.
- Use the curved lower surface idea for modal/bottom-sheet flows on mobile, but use ordinary panels on desktop and tablet POS layouts.
- Use a price plus primary button row for checkout, refund/cancel confirmation, and paid-order detail actions.
- Use icon facts only for compact status summaries. Do not replace essential order details with decorative icon rows.

## 5. Shared Typography

The image uses a rounded, modern sans-serif with deep navy headings and light gray body copy. The hierarchy is clear even though text is small.

Styling cues:

- Headings are dark navy, medium weight, and not overly large.
- Body copy is muted and compact.
- Prices and selected values are higher contrast.
- Active chip labels are white and bold enough to read on green.

Coffee POS application:

- Use `Inter` or the current system sans-serif stack.
- Make POS labels larger than the image where needed; live cashier use is less forgiving than a shopping mockup.
- Avoid tiny descriptive paragraphs in repeated cards. Use concise labels, prices, and modifier summaries.

## 6. Shared Shape, Depth, And Material

The image uses a layered material system: large rounded shells, pill controls, small cards, circular icon buttons, and soft shadows.

Styling cues:

- Large app shells have high radii.
- Cards have modest radii and cast a soft, close shadow.
- Pills are used for search, selected tabs, and primary checkout actions.
- Circular buttons are used for icon-only controls.
- Borders are almost invisible, relying on shadow and surface contrast.

Coffee POS application:

- Use `8px` radius for repeated menu/order/report cards so dense layouts stay crisp.
- Use larger radii for app shells, dialogs, and mobile bottom sheets.
- Keep shadows subtle. POS screens should feel clean, not layered like a marketing mockup.
- Use borders for tables and high-density areas where shadow alone is not enough.

## 7. Shared State Language

The reference image uses green for "selected", "favorite", "price", and "checkout". It does not show error or disabled states, so Coffee POS needs explicit additions.

State mapping:

- Green: selected, primary action, paid, success, available.
- Navy: important text, prices when not using green, queue numbers.
- Muted gray: descriptions, helper text, inactive icons.
- Red: validation errors, failed network requests, destructive confirmations.
- Amber: warnings such as unprinted paid order or payment not confirmed.
- Blue: informational states such as backend reconnecting.

Coffee POS application:

- Never use color alone. Pair state color with text, icon, or status label.
- Disabled controls should be visibly disabled and include a reason nearby when the cashier needs to recover.
- Error messages should be short, specific, and placed next to the failed action.
