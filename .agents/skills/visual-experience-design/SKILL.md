---
name: visual-experience-design
description: Design, implement, and visually verify polished web UI, interaction systems, cinematic scenes, and purposeful motion for products such as MysteryVerse. Use for new pages, redesigns, design systems, game-like interfaces, storytelling moments, animation engineering, and browser-based visual QA. Do not use for backend-only work, generic copywriting, or requests where visual behavior is irrelevant.
---

# Visual Experience Design

Use this skill to move from a feature description to a coherent product experience, not merely a decorated page. The required outcome is a usable interface whose visual language, information hierarchy, interaction feedback, scene construction, and motion all support the product's task and emotional intent.

## Operating Stance

- Read the existing product before inventing a visual direction. Preserve established tokens, components, typography, routing, data contracts, and interaction conventions unless the user explicitly requests a redesign.
- Treat the interface as a staged scene. Decide what the user must notice first, what is ambient, what is actionable, and what should remain hidden until the product state permits it.
- Use spectacle only where it carries meaning. A clue reveal, identity reveal, phase transition, or result moment can justify a cinematic sequence; routine navigation and form controls should remain quiet and fast.
- Do not equate a component library with a design system. Libraries supply primitives and implementation patterns; the product still needs hierarchy, tokens, copy, states, composition, and a point of view.
- When the repository has no application yet, deliver design specifications and implementation-ready contracts. Do not pretend that screenshots or runtime verification occurred.

## Trigger Conditions

Apply this skill when the task involves one or more of the following:

- Creating or redesigning a page, flow, dashboard, game interface, or interactive story surface.
- Defining visual tokens, typography, color, spacing, component states, themes, or responsive behavior.
- Designing atmospheric scenes, layered backgrounds, evidence boards, role reveals, maps, or narrative transitions.
- Adding hover, page, modal, list, text, gesture, or state-change motion.
- Choosing among CSS transitions, Motion, GSAP, Canvas, WebGL, or Three.js.
- Auditing an implemented interface with screenshots, browser interaction, accessibility checks, or motion performance checks.

Do not use this skill for API-only work, database design, prose editing with no interface artifact, or superficial code tasks where visual quality and interaction behavior cannot change.

## First Pass: Context Inventory

Before proposing a visual direction for a substantial page or redesign:

1. Locate the PRD, design tokens, theme files, component library, routing, state model, and relevant page code.
2. Identify the current technology stack and available animation/rendering dependencies. Do not add a dependency until the existing stack cannot express the required behavior cleanly.
3. Identify the user, task, current product phase, information visibility rules, and primary action.
4. Inventory available assets. Never substitute stock imagery, emoji, or decorative effects for domain-specific evidence when the interface needs to show an actual place, object, person, or state.
5. Record constraints that affect the experience: desktop/mobile support, reduced motion, privacy, timers, real-time updates, content safety, localization, and performance budget.

For MysteryVerse, also identify the mode: scripted murder mystery, werewolf, or immersive RPG. Read `references/mysteryverse-design-language.md` when the task is for this product.

## Design Workflow

### 1. Define the experience brief

Write a compact brief before implementation for any new page, major feature, or redesign:

- Product promise in one sentence.
- Target user and current task.
- Emotional target and required clarity level.
- Primary focal point and secondary information.
- What must remain persistently visible.
- What must never be revealed in the current state.
- One visual direction, with a short reason it belongs to this product.

For substantial work, structure the brief with `templates/page-design-spec.md`. Do not generate a separate document when the user asked for a direct code change; the brief can be included in the working notes or response.

### 2. Compose the information hierarchy

Define the page as three layers:

- **Operational layer:** current phase, timer, primary action, safety/state indicators, and required controls.
- **Interpretive layer:** narrative, clues, evidence, player/role context, relationships, outcomes.
- **Atmospheric layer:** environment, lighting, texture, ambient motion, sound cues, and transition space.

The operational layer must remain legible and reachable before the atmospheric layer is allowed to become visually dominant. On mobile, convert secondary panels to drawers, tabs, sheets, or explicit view switches without hiding the current primary action.

### 3. Establish the visual system

Read `references/visual-design-system.md` when creating or extending tokens, themes, typography, spacing, components, or responsive layouts.

Establish:

- Semantic color roles, including light/dark themes and game-state colors.
- Type families, size scale, line height, measure, and numeric/timer treatment.
- Spacing, grid, radius, border, elevation, icon, and density rules.
- Component variants and state coverage.
- Responsive composition, not just shrinking desktop panels.
- Accessibility constraints for contrast, focus, zoom, and reduced motion.

Use product-specific visual signals rather than a generic SaaS palette. MysteryVerse may use ink, paper, archive metal, evidence amber, discovery green, and danger red, but the exact palette must respond to the current design system and scene.

### 4. Build the scene, not a background

Read `references/cinematic-scene-design.md` when the interface includes environmental storytelling, evidence discovery, role reveal, maps, NPC scenes, or phase transitions.

Specify the scene in layers:

1. Environment and time/weather.
2. Background architecture or landscape.
3. Midground evidence, characters, objects, or path.
4. Foreground occlusion, framing, texture, or depth cue.
5. Lighting direction and color temperature.
6. Interaction focus and sightline.
7. Ambient motion and sound, if supported.
8. Scene state mapping and transitions.

Use real or generated bitmap assets when the subject must be inspected. Use CSS/Canvas/WebGL for atmosphere only when it improves depth, state communication, or performance. Do not place every game scene inside the same card shell.

### 5. Map interactions and states

Read `references/interaction-patterns.md` when designing controls, dialogs, drawers, forms, lists, tabs, loading, empty, error, reconnect, or game actions.

For each key action, define:

- Trigger and permitted state.
- Immediate visual acknowledgement.
- Pending/confirmed/failed result.
- Whether the action is optimistic or server-confirmed.
- Keyboard and focus behavior.
- Mobile equivalent.
- Recovery path for disconnect, timeout, conflict, or stale state.

In real-time game flows, separate local feedback from authoritative state. Never animate a success that has not been confirmed when the operation affects clues, votes, identity, elimination, inventory, or world state.

### 6. Specify motion as choreography

Read `references/motion-design-guidelines.md` before adding non-trivial animation. Use `templates/motion-spec.md` for a cinematic sequence, complex component transition, or game phase change.

Define:

- Narrative purpose and information revealed at each beat.
- Trigger, duration, easing, start/end state, and overlap.
- Interruptibility, replay behavior, and completion callback.
- Reduced-motion alternative.
- Performance tier and fallback.
- Sound cue, if the product supports sound.

Choose the smallest capable engine:

1. CSS transitions/animations for simple state and micro-interactions.
2. Motion for React layout, presence, gesture, and component choreography.
3. GSAP for long, precisely sequenced timelines or scroll-driven scenes.
4. Canvas for dense particles or custom 2D rendering.
5. Three.js/WebGL only when spatial depth or 3D interaction creates real product value.

Do not introduce a second animation system for one effect.

### 7. Implement inside the existing system

- Reuse tokens, primitives, layout shells, and utilities before creating new ones.
- Keep visual state separate from authoritative business state.
- Keep motion orchestration out of unrelated data logic.
- Give fixed-format controls stable dimensions so hover, labels, counters, and loading states cannot shift layout.
- Use semantic HTML and accessible primitives. Icon-only controls require an accessible name and visible tooltip when the meaning is not obvious.
- Respect `prefers-reduced-motion`; provide a static or low-motion path that still communicates state.
- Clean up timelines, observers, animation frames, listeners, textures, geometry, and render targets.
- Avoid continuous animation when the user is reading, entering text, voting, or comparing evidence.

### 8. Verify the rendered experience

Read `templates/visual-review-checklist.md` before declaring the work complete. Use a browser and screenshots when a runnable app exists.

Minimum verification:

- Desktop and mobile viewport screenshots.
- Primary task completed with mouse and keyboard.
- Loading, empty, error, pending, success, and reconnect states where applicable.
- No clipped text, overlap, accidental scroll traps, or controls pushed off-screen.
- Console errors and failed asset requests checked.
- Reduced-motion behavior checked.
- Animation smoothness and cleanup checked for heavy scenes.

If visual evidence cannot be produced, state that clearly and list the remaining runtime risk.

## Design Decision Rules

### Visual quality

- One clear focal point per viewport.
- Strong hierarchy before decoration.
- Readable text over atmospheric backgrounds; use controlled contrast and local backplates when necessary.
- Prefer a restrained number of materials and effects. Repeat a visual language instead of collecting unrelated effects.
- Do not make all pages dark purple, glassy, gradient-heavy, or card-heavy.
- Do not use cards inside cards as a page-layout strategy.
- Do not use emoji as a substitute for a professional icon system.
- Do not hide poor spacing or hierarchy under glows, noise, blurs, or gradients.

### Motion quality

- Animation must communicate arrival, causality, relationship, state change, attention, or emotion.
- The most important information is never delayed solely for spectacle.
- Use motion budgets: quiet UI, narrative moment, and cinematic set piece must not share the same intensity.
- Continuous ambient effects must be subtle, pausable where practical, and disabled under reduced motion.
- Use transform and opacity for high-frequency motion; avoid animating layout-triggering properties unless the interaction genuinely requires it.

### Narrative quality

- UI state and story state must agree. A clue cannot appear archived before it is acquired; a dead player cannot retain an active action surface; an unrevealed identity cannot leak through layout or motion.
- Private and public information must be visually distinguishable without relying on color alone.
- A major reveal should leave a persistent artifact or state change, such as a new archive entry, timeline node, relationship edge, or phase marker.
- Do not imply choices or actions that the rules engine cannot support.

## Reference Routing

Read only the references needed for the current task:

- `references/design-principles.md`: decision criteria, tradeoffs, research anchors, and review posture.
- `references/visual-design-system.md`: tokens, typography, color, spacing, components, themes, responsive rules.
- `references/cinematic-scene-design.md`: scene composition, lighting, layers, environmental storytelling, scene states.
- `references/motion-design-guidelines.md`: timing, easing, engine selection, choreography, reduced motion, cleanup, performance.
- `references/interaction-patterns.md`: controls, forms, overlays, feedback, game interactions, accessibility, reconnect.
- `references/anti-ai-slop.md`: common failure patterns and the hard pre-delivery audit.
- `references/mysteryverse-design-language.md`: brand direction and mode-specific rules for MysteryVerse.

Use templates when a structured artifact improves execution:

- `templates/page-design-spec.md`: substantial page or flow design.
- `templates/motion-spec.md`: major animation or phase transition.
- `templates/visual-review-checklist.md`: browser and visual acceptance.

Use examples only when the current problem is analogous:

- `examples/mystery-scene-example.md`: layered mystery environment and scene state.
- `examples/evidence-reveal-example.md`: clue discovery through archival state change.
- `examples/game-transition-example.md`: werewolf night-to-day phase transition.

## Output Requirements

For implementation work, the final response must include:

- What experience decision was made and why.
- Which files changed.
- Which states, breakpoints, and motion paths were implemented.
- What was verified in a browser, and what remains unverified.
- Any dependency, asset, content, or backend contract still required.

For design-only work, provide the same information as a concrete implementation brief, including tokens, layout behavior, motion contracts, and acceptance criteria. Do not present mock prose as if code or screenshots exist.

## Quality Gate

Do not mark the work complete if any of the following is true:

- The primary task or current game action is not visible and reachable at the target breakpoints.
- Text overlaps, clips, or becomes unreadably small.
- Loading, empty, error, pending, reconnect, or permission states are missing for a real workflow.
- The interface leaks private game information or visually implies an unconfirmed result.
- Motion hides essential information, blocks input, cannot be reduced, or is used as decoration without purpose.
- A visual effect has no fallback for low-performance devices or unavailable assets.
- The implementation introduces a heavy dependency without a documented experience reason.
- The result looks like a generic component-library demo and does not reflect the product's domain, content, or state.
