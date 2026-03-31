# Design System Document

## 1. Overview & Creative North Star
The Creative North Star for this design system is **"The Sentient Console."** 

This is not a traditional educational app; it is a high-performance terminal interface designed for the developer’s mind. It moves away from the "playful" tropes of language learning and instead adopts an aesthetic of precision, technical mastery, and AI-driven efficiency. 

To break the "standard template" feel, the system utilizes **Monolithic Asymmetry**. We favor wide, cinematic gutters, left-heavy information density, and interactive elements that feel like executable code blocks. The layout mimics a sophisticated IDE, where content isn't just displayed—it is "compiled." We replace rounded, friendly corners with sharp 0px radii to emphasize a brutalist, hardware-inspired rigidity.

---

## 2. Colors & Surface Philosophy

### The Palette
The color logic is rooted in high-contrast functionalism.
- **Primary (`#9cff93` / `#00fc40`):** The "Execute" color. Used for progress, success states, and primary actions. It represents the flow of data.
- **Secondary (`#fcaf00`):** The "Warning/Focus" color. Used for highlighting specific linguistic nuances, active selection states, or high-priority technical terms.
- **Tertiary (`#81ecff`):** The "System Info" color. Reserved for secondary data visualization and "AI Tutor" metadata.
- **Background (`#0a0e14`):** A deep, void-like navy that provides the canvas for the glowing terminal elements.

### The "No-Line" Rule
Traditional 1px borders are strictly prohibited for structural sectioning. To define layout boundaries, designers must use **Tonal Shifts**. A sidebar should be distinguished from the main viewport by shifting from `surface` to `surface-container-low`. We define space through volume, not outlines.

### Surface Hierarchy & Nesting
UI depth is achieved through **Nesting Levels**:
1.  **Base Layer:** `surface` (#0a0e14) – The fundamental background.
2.  **Sectional Layer:** `surface-container-low` (#0f141a) – Large layout blocks (e.g., the terminal input area).
3.  **Component Layer:** `surface-container` (#151a21) – Individual cards or modules.
4.  **Interactive Layer:** `surface-bright` (#262c36) – Active states or elevated modals.

### The "Glass & Gradient" Rule
Floating elements (like tooltips or temporary overlays) should utilize **Glassmorphism**. Use semi-transparent `surface-container-highest` with a `20px` backdrop-blur. For primary buttons, apply a subtle linear gradient from `primary` to `primary-container` at a 45-degree angle to give the "neon" a vibrating, physical energy.

---

## 3. Typography
The typography system balances the technical precision of Monospaced fonts with the high-end editorial feel of Space Grotesk.

*   **Display & Headline (Space Grotesk):** These are our "System Status" headers. Use `display-lg` for session milestones. Headlines should always be preceded by a `>` prompt character in `primary` to reinforce the terminal aesthetic.
*   **Body & Titles (Inter):** Chosen for its exceptional readability in dark mode. Inter provides a clean, neutral contrast to the aggressive display type. 
*   **Labels (Space Grotesk):** Used for "Metadata" (e.g., `// ATTEMPT_COUNTER`). Labels should always be in uppercase or preceded by double slashes `//` to mimic code comments.

---

## 4. Elevation & Depth

### The Layering Principle
Forget shadows; think of "stacked plates." To make a card feel "raised," do not use a drop shadow. Instead, place a `surface-container-highest` object on a `surface` background. The contrast in hex values provides all the necessary elevation.

### Ambient Glows
In lieu of traditional shadows, use "Status Glows." When a component is active (like a selected language card), apply a very soft, `8%` opacity outer glow using the `primary` color. This simulates the light emission of an old CRT monitor.

### The "Ghost Border" Fallback
If visual separation is failing in high-density areas, use a **Ghost Border**: 1px solid `outline-variant` at `15%` opacity. This should feel like a faint grid line on a blueprint, not a container wall.

---

## 5. Components

### Terminal Input (Text Fields)
- **Visuals:** Use `surface-container-lowest` for the fill. No border except for a 2px `primary` bottom-bar when focused.
- **Details:** The cursor should be a solid `primary` block, optionally blinking.
- **Prompt:** Every input must start with a `>` character.

### Segmented Progress Bars
- **Visuals:** Do not use continuous fills. Use discrete segments (the Spacing Scale `0.5` or `1` for gaps).
- **Logic:** Completed steps use `primary`. Current steps use `secondary`. Future steps use `surface-container-highest`.

### Interactive Cards
- **Visuals:** 0px border-radius.
- **Brackets:** Active cards are "wrapped" in terminal brackets: `[ Content ]`. The brackets should be part of the `:hover` and `:active` states, appearing as if the system is "selecting" that block of code.

### Buttons
- **Primary:** High-contrast `primary` background with `on-primary` text. Use a "Glitch" hover effect (slight 1px X/Y offset shift).
- **Ghost/Tertiary:** No background. Text wrapped in `[ ]` brackets. Use `primary` or `secondary` for the text color.

### Segmented Lists
- **Rule:** Forbid divider lines.
- **Execution:** Use `2.5` (0.5rem) vertical spacing between list items. Use a `surface-container-low` background on hover to highlight the row.

---

## 6. Do's and Don'ts

### Do
- **DO** use `//` for all helper text or secondary labels to maintain the developer vibe.
- **DO** use the `secondary` (Amber) color sparingly for "Critical Syntax" or "New Words" to ensure they pop against the neon green.
- **DO** lean into "Oversized Metadata." Displaying technical stats (e.g., `LATENCY: 14ms`, `LEVEL: B2`) adds to the immersion.

### Don't
- **DON'T** use rounded corners (`border-radius: 0` is the law).
- **DON'T** use standard "Success/Error" colors (Red/Green) in a way that clashes with the `primary` neon. Use `error` (#ff7351) specifically for syntax failures, but keep it muted compared to the `primary` neon.
- **DON'T** use heavy drop shadows. They muddy the "Sentient Console" aesthetic. Stick to tonal layering and glow.
- **DON'T** use generic icons. Use ASCII-inspired icons or simplified monolinear technical symbols.