# Visual Review Checklist

Use this checklist after implementation and before declaring visual work complete. Record actual browser/viewport evidence. Do not mark runtime items complete if the app could not be run.

## Review metadata

- Page/flow:
- Mode:
- Commit/build:
- Reviewer:
- Desktop viewport(s):
- Mobile viewport(s):
- Browser(s):
- Reduced-motion state:
- Sound state:
- Known limitations:

## 1. Primary task

- [ ] The page's current question is obvious within the first viewport.
- [ ] The primary action is visible without hunting.
- [ ] The primary action is reachable on mobile.
- [ ] Current phase/timer/status remains visible.
- [ ] Destructive or irreversible actions state the consequence.
- [ ] The flow completes with mouse/touch and keyboard.

## 2. Composition

- [ ] One clear focal point per viewport.
- [ ] Operational UI is more prominent than atmospheric decoration.
- [ ] Secondary panels are visually subordinate.
- [ ] Page sections are not all styled as floating cards.
- [ ] Cards are not nested inside decorative cards.
- [ ] Negative space is intentional.
- [ ] Typography and scale carry the hierarchy, not just shadows/glows.
- [ ] The page does not resemble a generic SaaS/dashboard template.
- [ ] The active MysteryVerse mode is recognizable from composition.

## 3. Typography and content

- [ ] Text fits its container at all checked widths.
- [ ] Long Chinese labels/names wrap cleanly.
- [ ] Buttons, tabs, timers, and counters do not resize on hover/loading.
- [ ] Body measure is readable.
- [ ] Numeric/timer values use stable/tabular treatment.
- [ ] Copy names real actions and consequences.
- [ ] Empty/error copy offers a legal next step.
- [ ] No fake content is presented as real product state.

## 4. Color and materials

- [ ] Semantic colors are used consistently.
- [ ] The viewport is not dominated by one hue family.
- [ ] Focus is visible against every surface.
- [ ] Text contrast passes against the final composite.
- [ ] State meaning is not carried by color alone.
- [ ] Private/public scope is labeled, not only tinted.
- [ ] Material effects have a domain purpose.
- [ ] Gradients/blur/noise do not hide weak hierarchy.

## 5. Scene and imagery

- [ ] Image subject is inspectable when the task requires it.
- [ ] Background focal crop works on mobile.
- [ ] Foreground, midground, and background layers read distinctly.
- [ ] Lighting direction is coherent.
- [ ] Effects are absent from reading areas or masked appropriately.
- [ ] Missing/failed assets have a designed fallback.
- [ ] Scene state matches authoritative game state.
- [ ] Private content is absent from public scene layers.
- [ ] Major reveal leaves a persistent artifact/state.

## 6. Interaction and state

- [ ] Hover, focus, pressed, loading, disabled, success, and error states exist.
- [ ] Pending game-critical action does not show false success.
- [ ] Reconnect state blocks stale/duplicate submission.
- [ ] Stale/conflict state offers recovery.
- [ ] Dialogs/sheets have titles and logical close behavior.
- [ ] Tooltips are available for unfamiliar icon-only controls.
- [ ] Toasts are not the only feedback for critical actions.
- [ ] Empty/loading/error states preserve page context.

## 7. Motion

- [ ] Every significant animation has a stated purpose.
- [ ] Essential information is not delayed for spectacle.
- [ ] Quiet UI does not use cinematic timing.
- [ ] Layout shifts on hover are absent unless semantically required.
- [ ] Repeated trigger/interruption does not leave stale visual state.
- [ ] `prefers-reduced-motion` provides a meaningful alternative.
- [ ] Continuous effects pause offscreen/hidden where practical.
- [ ] Animation cleanup runs on unmount/route change.
- [ ] Private data does not appear in transition clones.

## 8. Responsive

- [ ] Desktop composition is intentional at 1024px+.
- [ ] Medium layout avoids compressed three-column panels.
- [ ] Mobile uses recomposition, tabs, sheets, or drawers.
- [ ] Current primary game action remains fixed/visible.
- [ ] Safe-area insets respected.
- [ ] No accidental horizontal scroll.
- [ ] Buttons and hit targets do not overlap.
- [ ] Text does not clip due to fixed heights.
- [ ] Scene crop and UI overlay remain coherent.

## 9. Accessibility

- [ ] Heading/landmark structure is logical.
- [ ] Keyboard can complete the primary flow.
- [ ] Focus order matches reading/action order.
- [ ] Modal focus trap and focus return work.
- [ ] Icon-only controls have accessible names.
- [ ] Canvas/spatial UI has a list or keyboard alternative.
- [ ] No essential information depends on sound or motion.
- [ ] Flash/shake/intense sound can be disabled or avoided.
- [ ] Browser zoom/text resizing does not break the flow.

## 10. Technical and performance

- [ ] Console has no relevant errors.
- [ ] Network panel shows no unexpected failed assets.
- [ ] High-frequency animation uses transform/opacity where practical.
- [ ] Particle/blur/canvas work is bounded.
- [ ] Low-performance fallback exists for heavy scene work.
- [ ] No duplicate timelines, observers, loops, or event listeners accumulate.
- [ ] No new heavy dependency was added without a documented reason.
- [ ] Images have stable dimensions and appropriate responsive sources.

## 11. Visual evidence

Attach or reference:

- Desktop screenshot:
- Mobile screenshot:
- Key intermediate/animation frame:
- Reduced-motion screenshot:
- Error/empty/reconnect screenshot:

## 12. Final decision

- Pass:
- Needs revision:
- Blocking issues:
- Residual risk:
- Follow-up:
