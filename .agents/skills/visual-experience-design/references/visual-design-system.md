# Visual Design System

Use this reference to define or extend a product's visual language. Tokens listed here are a strong default for MysteryVerse, not a mandate to overwrite an existing system. Preserve the repository's current tokens when they already solve the same role.

## 1. System layers

Keep the design system in four layers:

1. **Primitive tokens:** raw values such as colors, type sizes, spacing, radii, and durations.
2. **Semantic tokens:** purpose-based aliases such as `surface-canvas`, `text-muted`, `state-danger`, and `evidence-accent`.
3. **Component contracts:** anatomy, sizes, states, and behavior for buttons, inputs, tabs, dialogs, cards, and game controls.
4. **Page patterns:** shells, grids, focal compositions, and responsive transformations.

Agents should consume semantic tokens in feature code. Primitive tokens belong in theme files.

## 2. Color

### Base palette

Use a mixed temperature palette with deliberate hierarchy:

| Role | Candidate | Character |
| --- | --- | --- |
| Canvas dark | `#0B0D10` | near-black, not blue-purple |
| Raised dark | `#14181D` | panel and scene frame |
| Elevated dark | `#1D232A` | active tool or selected context |
| Canvas light | `#F1EDE4` | paper-like reading/creator surface |
| Paper | `#E6DFD2` | dossier, evidence, script |
| Ink | `#171A1D` | bright-theme text |
| Bone | `#E8E3D8` | dark-theme primary text |
| Ash | `#A7AAA8` | secondary text |
| Metal | `#6E7778` | borders, inactive icons |
| Evidence amber | `#C7783F` | discovered clue, archive seal |
| Discovery green | `#4D8273` | valid connection, new knowledge |
| Danger red | `#B34D4B` | destructive action, danger, accusation |
| Moon silver | `#B7C3CC` | werewolf public phase metal |
| Faction oxblood | `#713B43` | hidden faction signal, used sparingly |
| Moss | `#63775B` | RPG environment and exploration |
| Copper | `#A46F4F` | RPG artifacts, route markers |

Do not use all accents in one viewport. Establish one dominant accent and one state accent.

### Semantic tokens

Minimum semantic set:

```css
:root {
  --color-canvas: #0b0d10;
  --color-surface: #14181d;
  --color-surface-raised: #1d232a;
  --color-surface-inset: #090b0e;
  --color-text-primary: #e8e3d8;
  --color-text-secondary: #a7aaa8;
  --color-text-disabled: #696d6e;
  --color-border-subtle: #2c333a;
  --color-border-strong: #596166;
  --color-focus: #e2b36d;
  --color-action-primary: #d4a25c;
  --color-action-primary-hover: #e6b973;
  --color-action-ink: #171a1d;
  --color-state-success: #4d8273;
  --color-state-warning: #c99743;
  --color-state-danger: #b34d4b;
  --color-state-info: #6e93a6;
  --color-evidence: #c7783f;
  --color-public: #8c9a9d;
  --color-private: #d4a25c;
  --color-secret: #713b43;
}
```

Define `data-theme="paper"`, `data-theme="archive"`, or the project's existing theme mechanism. Theme switching must update semantic tokens, not page-specific selectors.

### Usage rules

- Body text must meet contrast requirements against its actual rendered backdrop.
- Danger, success, and private/public meaning cannot rely on color alone; add labels, icons, line style, containment, or copy.
- Gradients are allowed only when they model light, depth, atmosphere, or material. Do not use them as a substitute for composition.
- Texture, grain, and noise should be low-frequency and subordinate to text.
- The default game shell should not become a purple-blue neon palette. If a scene needs violet light, treat it as scene lighting rather than the product's global brand color.

## 3. Typography

### Families

Prefer a two-family system:

- **Narrative/display:** `"Noto Serif SC", "Source Han Serif SC", "Songti SC", Georgia, serif`.
- **UI/body:** `"Noto Sans SC", "Source Han Sans SC", "PingFang SC", Inter, system-ui, sans-serif`.
- **Technical/data:** `"IBM Plex Mono", "SFMono-Regular", Consolas, monospace` for timers, IDs, codes, and tabular data.

Use local/system fallbacks unless the project already self-hosts fonts. Do not load a large font family only for one decorative heading.

### Scale

Use responsive CSS variables but do not scale font size with viewport width.

| Token | Size / line height | Use |
| --- | --- | --- |
| `text-display` | 56/60 desktop, 38/42 mobile | rare scene title or result reveal |
| `text-h1` | 40/46 desktop, 32/38 mobile | page or major scene heading |
| `text-h2` | 28/34 | section heading |
| `text-h3` | 22/28 | panel heading |
| `text-body-lg` | 18/30 | narrative reading |
| `text-body` | 16/26 | default reading |
| `text-ui` | 14/20 | controls and dense UI |
| `text-caption` | 12/18 | metadata and labels |
| `text-micro` | 11/16 | only for non-critical metadata |

Use `letter-spacing: 0` for normal UI. Small caps/labels may use positive tracking up to `0.08em`. Never use negative letter spacing.

### Reading measure

- Narrative text: 45-72 characters per line.
- Dense side panels: 30-48 characters.
- Dialog max width: 42-58rem depending on content.
- Use hanging punctuation only when supported and visually checked.

## 4. Spacing and grid

Use a 4px base scale:

```text
0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128
```

Recommended aliases:

```css
:root {
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;
  --space-16: 64px;
  --space-20: 80px;
}
```

Desktop application shell:

- 12-column content grid with 24px gutters.
- Max reading width for ordinary pages: 1440-1600px.
- Gameplay surfaces may use edge-to-edge scene composition with constrained operational overlays.
- Sidebars must have stable min/max widths and internal scrolling.

Mobile:

- 4-column grid with 16px side padding.
- Respect safe-area insets.
- Keep the current primary game action visible.
- Convert secondary panels into tabs, drawers, or explicit views; do not compress desktop columns into unreadable slivers.

## 5. Radius, border, elevation

Use a material hierarchy rather than making every panel float.

```css
:root {
  --radius-xs: 2px;
  --radius-control: 6px;
  --radius-card: 8px;
  --radius-overlay: 12px;
  --border-hairline: 1px;
  --border-emphasis: 2px;
  --shadow-card: 0 1px 0 rgb(255 255 255 / 0.04),
                 0 8px 24px rgb(0 0 0 / 0.18);
  --shadow-overlay: 0 20px 64px rgb(0 0 0 / 0.42);
}
```

Rules:

- Cards use 8px radius or less unless an existing design system requires otherwise.
- Do not add both a border, strong shadow, glass blur, and gradient to the same ordinary panel.
- Dossier/evidence paper may use clipped corners, perforations, stamps, or fold cues instead of rounded-card language.
- Overlays can use larger radius than cards; repeated content should not.
- Focus rings require clear contrast and must not be clipped by overflow.

## 6. Icons

- Use one coherent icon family already available in the project, preferably Lucide when the project uses React and has no stronger system.
- Icons carry a familiar concept; text carries nuance.
- Icon-only controls need an accessible name and a tooltip for unfamiliar actions.
- Do not use emoji as interface iconography.
- Keep optical sizes consistent: 16px inline, 18-20px controls, 24px navigation/emphasis.
- Decorative icons should be hidden from assistive technology.

## 7. Themes and seasonal surfaces

Design theme switching through a small set of surface modes:

- **Archive Night:** dark game shell, low ambient light, warm evidence accents. Best for investigation, werewolf, and focused gameplay.
- **Paper Daylight:** light editorial surface for scripts, rules, creator tools, profiles, and long reading. Use ink and paper rather than pure white.
- **Scene Mode:** the asset/lighting palette of an RPG or scripted environment can override ambient colors within a bounded scene region, while controls continue using semantic state tokens.
- **High Contrast:** stronger borders, reduced texture, no translucent text backplates, and simplified scene effects.

Never make "theme switch" a global color inversion that destroys scene assets and state colors.

## 8. Component contracts

### Buttons

Define visual hierarchy by action consequence:

- Primary: current phase's main confirmed action; one per operational region.
- Secondary: alternative valid action.
- Quiet: navigation or low-priority utility.
- Destructive: accusation, elimination confirmation, deletion, rollback.
- Disabled: unavailable rule action; explain why nearby or on focus/tap.

Every button needs hover, focus-visible, pressed, loading, success/failure acknowledgement, and disabled behavior. Loading must not resize the control.

### Inputs and text entry

- Visible label, not placeholder-only labeling.
- Error text connected to the field.
- Preserve user input through transient failures.
- For free-form RPG actions, show examples or suggested legal intents without pretending they are the only options.
- Autocomplete and spellcheck should be deliberate for Chinese narrative input.

### Segmented controls and tabs

- Use for mutually exclusive views or modes.
- Keep indicator motion short and subordinate.
- On mobile, avoid squeezing more than four short labels; use a menu or scrollable tab rail if necessary.
- Preserve active-state semantics and keyboard behavior.

### Cards

Use cards for repeated objects, modals, and genuinely bounded tools. Do not make every page section a card.

- Stable image aspect ratio.
- Stable title area and metadata rows.
- Hover may lift, reveal a border, or expose secondary actions, but cannot shift surrounding content.
- Do not nest a card grid inside a decorative card.

### Dialogs, sheets, and drawers

- Use modal semantics and focus management.
- Trap focus only for true modals and return focus on close.
- Title is always present for assistive technology.
- A cinematic overlay still needs an escape/close path unless the rules make dismissal impossible.
- Use drawers for contextual side panels on smaller screens, not as the default desktop decoration.

### Status, progress, and timers

- Pair text with visual state.
- Timers use tabular numerals and stable width.
- Warning states appear before expiry, not only at zero.
- Progress for AI generation must distinguish queued, parsing, generating, validating, and failed.
- A long-running process must have a truthful "still working" state and safe retry path.

### Evidence and dossier components

Use domain-specific anatomy:

- Evidence header: source, acquisition context, visibility scope.
- Evidence body: excerpt, image/diagram, annotations.
- Evidence metadata: timestamp, location, related people, credibility marker.
- Actions: inspect, compare, pin to theory, request DM clarification, archive.

Use paper, tabs, stamping, thread, ink, and metal selectively. Do not style every evidence item as a glass card.

## 9. Layout patterns

### Editorial/creator page

- Constrained reading column plus a tool rail or inspector.
- Strong page title and section rhythm.
- Dense forms grouped by decision, not object count.
- Sticky save/publish status only when it does not obscure content.

### Operational game page

- Persistent phase status and legal action frame.
- Primary narrative/detail region in the center.
- Private context in a side panel or mobile tab.
- Public communication remains discoverable without competing with the immediate action.

### Evidence board

- Spatial arrangement must encode relationships, not random masonry.
- Use line, pin, label, and grouping styles to distinguish fact, hypothesis, and contradiction.
- Provide a list/accessible alternative to freeform spatial navigation.

### Werewolf ring

- Seat geometry should make turn, vote, death, and speaking state obvious.
- Night actions use role-private panels; the public ring remains readable but subdued.
- Vote outcomes animate from actual vote events, not decorative guesswork.

### RPG world

- Scene narrative is dominant; map, party, task, inventory stay one action away.
- Public world events and personal events must be visually distinct.
- Travel, interaction, and dialogue preserve scene context.

## 10. Responsive behavior

Define breakpoints by composition change, not device names:

```text
compact: < 640px
medium: 640-1023px
wide: 1024-1439px
ultrawide: >= 1440px
```

At each breakpoint, state:

- Which panels remain visible.
- Which become tabs/drawers.
- Which content reorders.
- Which content is intentionally deferred.
- Where the primary action sits.
- What happens to background focal crop and safe areas.

Do not use fixed pixel layouts that clip Chinese labels, timers, or icon+text controls. Use `minmax`, `clamp` for dimensions where appropriate, and stable control heights.

## 11. Accessibility baseline

- Keyboard access for the primary flow.
- Visible focus states that survive scene overlays.
- Logical headings and landmarks.
- Dialog title, focus trap, escape, and focus return.
- Text contrast checked against the final rendered backdrop.
- 44px minimum touch target where practical.
- No information conveyed only through color or motion.
- Respect browser zoom and text resizing.
- Provide a reduced-motion path.
- Provide captions/transcripts or a text equivalent for meaningful sound cues.
