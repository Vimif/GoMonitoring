## 2024-05-22 - [Invisible Spinners on Loading State]
**Learning:** Using `color: transparent` to hide text on buttons during loading states inadvertently hides pseudo-element spinners if they rely on `currentColor`.
**Action:** Define a specific CSS variable (e.g., `--btn-spinner-color`) for button variants to ensure the spinner remains visible while the text is hidden.
