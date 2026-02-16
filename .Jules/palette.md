## 2024-05-22 - [Invisible Spinners on Loading State]
**Learning:** Using `color: transparent` to hide text on buttons during loading states inadvertently hides pseudo-element spinners if they rely on `currentColor`.
**Action:** Define a specific CSS variable (e.g., `--btn-spinner-color`) for button variants to ensure the spinner remains visible while the text is hidden.

## 2026-02-12 - [Refactoring Machine Cards]
**Learning:** Nesting interactive buttons inside an anchor card wrapper is invalid HTML and inaccessible.
**Action:** Refactor to use a container div, link the primary title, and keep action buttons separate with aria-labels.

## 2026-03-24 - [Accessible Toggle Buttons]
**Learning:** Interactive toggles (like password visibility) must preserve natural tab order (no `tabindex="-1"`) and update `aria-label` dynamically to reflect state.
**Action:** Use `aria-label` to describe the button's action and update it via JS when the state changes.
