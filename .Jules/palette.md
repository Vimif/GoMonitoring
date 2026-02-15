## 2024-05-22 - [Invisible Spinners on Loading State]
**Learning:** Using `color: transparent` to hide text on buttons during loading states inadvertently hides pseudo-element spinners if they rely on `currentColor`.
**Action:** Define a specific CSS variable (e.g., `--btn-spinner-color`) for button variants to ensure the spinner remains visible while the text is hidden.

## 2026-02-12 - [Refactoring Machine Cards]
**Learning:** Nesting interactive buttons inside an anchor card wrapper is invalid HTML and inaccessible.
**Action:** Refactor to use a container div, link the primary title, and keep action buttons separate with aria-labels.

## 2026-02-15 - [Password Visibility Toggle Accessibility]
**Learning:** Password toggle buttons often have `tabindex="-1"` to prevent tabbing, which excludes keyboard users. Dynamic `aria-label` updates are crucial for screen readers to understand the current state.
**Action:** Always ensure interactive elements are in the tab order and use dynamic ARIA attributes for state changes.
