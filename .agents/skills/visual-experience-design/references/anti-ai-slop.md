# Anti-AI-Slop Design Standard

Use this reference for every visually significant task. "AI slop" is not merely a palette problem; it is a lack of product reasoning expressed through repeated visual defaults, weak hierarchy, generic content, and effects that do not change understanding.

## 1. Core test

For every major visual decision, answer:

1. What product state or user need does this express?
2. What would be lost if it were removed?
3. Is it derived from this specific domain or from a reusable template?
4. Does it still work on mobile, without motion, and without the best-case asset?

If the answers are vague, simplify or redesign.

## 2. Prohibited default patterns

Do not ship these without a specific product reason:

- Purple-blue gradient backgrounds on every product.
- Glassmorphism applied to most panels.
- Rounded card islands for every page section.
- Identical icon badges above every heading.
- Decorative orb/bokeh/blob backgrounds.
- Floating gradient borders on every card.
- One oversized marketing hero on an operational page.
- A dashboard where every module has equal visual weight.
- Multiple fonts and accent colors used to create "premium" feel.
- Emoji used as the product icon system.
- Generic stock images of people pointing at screens or vague atmospheric textures.
- Background video that does not reveal a product, place, or state.
- Repeating entrance animations on every card and section.
- Parallax or scroll hijacking for ordinary information.
- Continuous shimmer, particles, glow, marquee, or cursor effects behind reading text.
- Cards inside cards inside cards.
- Modal-on-modal flows when inline disclosure or a dedicated page would be clearer.
- Fake statistics, fake activity, fake testimonials, or placeholder content presented as real product data.

## 3. Visual hierarchy audit

Check:

- Is there one dominant focal point per viewport?
- Does the eye know where to go after the first second?
- Are operational actions more prominent than atmospheric decoration?
- Are secondary panels quieter than the primary region?
- Is type scale doing more work than shadows and borders?
- Is spacing grouping content by user decision?
- Are there too many items competing at the same contrast and size?
- Does each section need to be a separate surface?

A page with weak hierarchy cannot be rescued by adding gradient, blur, or animation.

## 4. Typography audit

- Body text is readable at the actual rendered backdrop.
- Chinese text has sufficient line height and is not forced into Latin-tight tracking.
- Headings are sized to their container.
- Long labels wrap cleanly.
- Button, tab, and counter text does not clip or resize the control.
- Numbers use tabular figures where comparison or timers require it.
- All-caps or letter-spaced labels are limited to short metadata.
- Decorative serif type is not used for dense controls or tiny UI copy.
- No negative letter spacing.
- No viewport-width font scaling.

## 5. Color audit

- The page is not dominated by variations of one hue.
- Purple, beige, dark blue, or brown-orange palettes are not used as automatic "premium" defaults.
- State colors have semantic roles and do not double as brand decoration.
- Text contrast is checked against the final composite.
- Accent color is not sprayed across every icon, border, and glow.
- Private/public/danger states use more than color.
- Scene lighting can be colorful without forcing the whole application shell into that color.

## 6. Surface audit

- Not every section is a card.
- Cards are used for repeated objects, modals, and bounded tools.
- Card radius is 8px or less unless an existing system requires otherwise.
- Panels use either border, shadow, material, or tonal difference as their primary separation, not all four.
- Repeated cards share anatomy but not identical content.
- Hero or page-level content is not trapped inside a decorative card.
- Overlays are visually justified and do not merely float above an already floating layout.

## 7. Content audit

- Copy names real actions and states.
- Labels are specific to MysteryVerse or the current product, not generic "Unlock potential".
- No feature-explainer copy inside the product unless it is necessary onboarding.
- No instructions that describe the interface instead of helping the task.
- Error copy states consequence and recovery.
- Empty copy offers a legal next step.
- Evidence, role, and scene content matches actual state.
- Privacy scope is explicit.

## 8. Motion audit

- Motion has a contract, not just a duration.
- It does not restart on every unrelated render.
- It does not delay essential information.
- It has a reduced-motion path.
- It has a cleanup path.
- It does not use five effects where one reveal would communicate better.
- It does not turn every hover into a spatial transformation.
- It does not use looping animation behind reading or decision areas.
- It does not imply success before authority confirms.

## 9. Responsive audit

- Mobile is a recomposed experience, not a compressed desktop.
- Primary action remains visible.
- No text is clipped by fixed heights.
- No horizontal scroll unless intentional and clearly signaled.
- Drawers and tabs retain titles and close behavior.
- Background focal crop still carries the subject.
- Touch targets do not overlap.
- Safe areas are respected.
- Scene and UI layers do not hide each other.

## 10. Accessibility audit

- Keyboard reaches the primary flow.
- Focus is visible over all surfaces.
- Dialogs have titles and manage focus.
- Icon-only controls have names and tooltips.
- State is not expressed only by color, motion, or sound.
- Reduced motion works.
- Flashing, shake, and intense audio can be disabled or avoided.
- Images that convey evidence have meaningful alternatives.
- Canvas/spatial interactions have a list or keyboard equivalent.

## 11. Performance audit

- The page does not load a heavy animation library for one fade.
- High-frequency animation avoids layout-triggering properties.
- Particle counts and blur areas are bounded.
- Offscreen effects pause.
- Images have responsive sources and stable dimensions.
- Animation instances clean up.
- Low-performance fallback is defined.
- The experience remains understandable if an asset fails.

## 12. Domain-specific slop

### MysteryVerse

Avoid:

- Every script becoming the same black/purple dashboard.
- "Evidence" shown as generic rounded cards with no source or visibility.
- Werewolf seats treated as a generic grid with role data visible to everyone.
- RPG scenes rendered as a background image plus ordinary chat panel.
- Every clue reveal using the same modal.
- Cinematic motion that exposes private information.
- Status text hidden by atmosphere.
- Decorative rain over unreadable body copy.
- Fake detective props unrelated to actual evidence.

Require:

- Public/private/system visibility in the information architecture.
- Mode-specific composition.
- Persistent state after reveals.
- Scene layers that reflect actual story state.
- A clear distinction between rule-confirmed events and AI narration.

## 13. The "remove one thing" rule

Before delivery, remove one non-essential decorative element from the strongest focal region. If the design improves, it was slop. Then inspect whether another element can be simplified.

## 14. Final hard gate

Do not approve the design if:

- The first screen could belong unchanged to an unrelated SaaS product.
- The only product identity is a logo and accent color.
- The visual hierarchy is carried mainly by cards and shadows.
- A generic illustration communicates the subject more than the real product, place, object, or state.
- The animation could be removed without any loss of state meaning and still dominates the experience.
- It cannot be understood with reduced motion, sound off, and a failed background asset.
