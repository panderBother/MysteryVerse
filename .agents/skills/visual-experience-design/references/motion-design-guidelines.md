# Motion Design Guidelines

Use this reference for interaction feedback, page and component transitions, narrative reveals, game phase changes, ambient scene motion, and animation engineering. Motion must clarify state and causality before it entertains.

## 1. Motion budget

Assign every animation to one intensity tier:

| Tier | Purpose | Typical duration | Examples |
| --- | --- | --- | --- |
| Quiet UI | acknowledgement and continuity | 80-220ms | hover, press, focus, tab indicator, tooltip |
| Structural | layout, overlay, panel, and route change | 180-420ms | drawer, dialog, evidence panel, page transition |
| Narrative moment | disclose or transform meaning | 500-1400ms | clue acquisition, role reveal, relationship link |
| Cinematic set piece | phase or world transition | 1.2-3.5s | night to day, location travel, final result |

Do not let routine controls use narrative or cinematic timing. Do not stack multiple high-intensity sequences without recovery time.

## 2. Animation contract

Every non-trivial animation must specify:

- Trigger.
- Business purpose.
- Information available before, during, and after.
- Initial and final state.
- Duration and easing per beat.
- Overlap, stagger, and delay.
- Interruptibility.
- Repeat behavior.
- Completion callback and failure behavior.
- Reduced-motion alternative.
- Performance fallback.
- Sound/haptic behavior.

Use `templates/motion-spec.md` for substantial sequences.

## 3. Easing

Use a small vocabulary:

```css
:root {
  --ease-standard: cubic-bezier(0.2, 0, 0, 1);
  --ease-enter: cubic-bezier(0.16, 1, 0.3, 1);
  --ease-exit: cubic-bezier(0.4, 0, 1, 1);
  --ease-emphasis: cubic-bezier(0.34, 1.2, 0.64, 1);
  --ease-linear: linear;
}
```

- `standard`: layout, movement, and state change.
- `enter`: objects arriving with quick initial progress and a soft settle.
- `exit`: objects leaving without prolonged linger.
- `emphasis`: single reveal or decisive stamp; use sparingly.
- `linear`: continuous ambient loops, progress rotation, and scroll-linked mapping.

Avoid bounce for serious information. Small overshoot can suit game tokens or card flips; it should never delay reading or produce nausea.

## 4. Duration guidance

| Interaction | Default |
| --- | --- |
| Button press | 80-120ms |
| Hover/focus | 120-180ms |
| Tooltip | 120-180ms |
| Dropdown/popover | 140-220ms |
| Tab indicator | 180-260ms |
| Dialog | 220-360ms |
| Drawer | 260-400ms |
| Page content reveal | 300-600ms |
| List stagger | 30-70ms per item, cap total |
| Clue card flip | 420-700ms |
| Archive insertion | 650-1000ms |
| Phase transition | 1.2-3.0s |

Durations scale with spatial distance and semantic weight, not viewport width.

## 5. Engine selection

### CSS

Use for:

- Hover, focus, and press.
- Simple enter/exit.
- Opacity, translation, scale, rotation.
- Skeleton shimmer.
- Small color transitions.
- Ambient loops with no orchestration.

Use `transition` when state toggles between two visual states. Use `@keyframes` when the same deterministic sequence repeats or has no interactive interruption.

### Motion for React

Use for:

- Layout transitions.
- Presence enter/exit.
- Shared-element-like continuity.
- Gesture-driven sheets and drawers.
- Staggered React component choreography.
- Reversible animation with state synchronization.

Prefer `MotionConfig reducedMotion="user"` or an equivalent project-level policy. Use layout animation only when parent/child movement is semantically useful.

### GSAP

Use for:

- Long timelines with precise labels, overlap, and callbacks.
- Complex SVG path or morph sequences.
- Scroll-linked storytelling.
- Choreography that must remain independent of React render frequency.

Create the timeline in a stable effect scope, kill it on cleanup, and avoid creating duplicate ScrollTriggers on every render.

### Canvas

Use for:

- Dense particles.
- Custom maps and relationship graphs with many nodes.
- Procedural weather or drawing.
- Pixel inspection inside a scene.

Keep the canvas focused on rendering; keep semantic content and controls in the DOM.

### Three.js/WebGL

Use only when:

- Spatial depth or camera movement is part of the task.
- A 3D object must be examined or manipulated.
- A scene cannot achieve the required physical continuity in 2D.

Do not use 3D for ordinary card tilt, text animation, or a hero that a bitmap/layered CSS would serve better.

## 6. Choreography principles

### Establish, then move

Let the eye register the state before adding secondary motion. Do not animate text, glow, particles, and controls simultaneously from frame zero.

### One dominant action per beat

Each beat should have one main change:

1. Environment responds.
2. Object arrives.
3. Information becomes readable.
4. Meaning is emphasized.
5. Result persists elsewhere.

### Reveal in reading order

- Heading before body.
- Identity before private detail.
- Evidence before interpretation.
- Outcome before explanation.
- Primary action after authority is ready.

### Preserve spatial continuity

If an item moves into an archive, use a continuous object or shared origin. Avoid crossfading unrelated layouts when continuity is the meaning.

### End in a stable state

The final frame should be the actual UI, not a transient overlay that then snaps. Clean up animation props, transforms, and temporary clones.

## 7. Interaction patterns

### Press and confirmation

- React immediately to pointer/keyboard press.
- Pending action must lock unsafe duplicate submission without a giant spinner.
- Confirmed state changes once authoritative data arrives.
- Rejected action returns to a stable state and explains why.

### Modal and drawer

- Animate the container and backdrop independently.
- Focus moves after the container is usable.
- Do not animate focus rings across the screen.
- Closing returns focus to the invoking control.
- Backdrop opacity should not make essential confirmation text unreadable.

### Lists and evidence grid

- Stagger only items in the current viewport.
- Cap total stagger duration.
- Do not replay all list animation on every filter unless the reordering is meaningful.
- Use layout animation to communicate moved or pinned evidence.

### Tabs

- The indicator may travel, but content should appear quickly.
- Avoid a long horizontal slide that hides the destination text.
- Keep panel height changes from shifting the current action unexpectedly.

### Loading

- Skeleton structure must approximate the final content.
- Streaming narrative should place and style text before tokens arrive.
- Long tasks show stage and elapsed status.
- Indeterminate ambient motion stays subtle.

## 8. Narrative and game sequences

### Evidence acquisition

Recommended beat structure:

1. Focus scene locally; dim non-target regions.
2. Acquire or flip evidence object.
3. Reveal readable evidence.
4. Mark keywords or connections.
5. Move/pin into archive.
6. Update evidence count and persistent entry.
7. Restore controls.

Do not animate interpretation before the player has seen the evidence. Do not reveal hidden metadata in the acquisition preview.

### Role reveal

1. Confirm any required user gesture.
2. Seal or obscure the role area.
3. Reveal faction/role in a single readable beat.
4. Show permitted knowledge and next action.
5. Keep role status persistently accessible after the sequence.

Private identity should never appear in a transition clone visible to other players or in a shared screen region.

### Phase transition

- Signal the phase change through one environmental system plus one operational system.
- Keep emergency controls and reconnect status accessible.
- Resume only after the authoritative phase is ready.
- Provide a reduced-motion variant that changes text, color, and status without full-screen movement.

## 9. Reduced motion

Provide a meaningful alternative, not simply `animation: none`.

Reduced-motion behavior:

- Remove parallax, shake, large translations, scale springs, and continuous particles.
- Replace flips with immediate side change or crossfade.
- Replace scene travel with a brief cut or opacity change.
- Preserve opacity/color/status transitions that improve comprehension.
- Keep timers, focus, progress, and confirmation visible.
- Skip cinematic delays and use minimum necessary reading pauses.

Implement in both CSS and JavaScript:

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    scroll-behavior: auto !important;
  }
}
```

Do not globally remove all transitions because focus or state feedback may depend on them. Apply reduced variants per interaction.

Use a runtime preference when Motion/GSAP owns the animation:

```tsx
const reduceMotion = useReducedMotion();
const transition = reduceMotion
  ? { duration: 0.12 }
  : { type: "spring", stiffness: 320, damping: 32 };
```

## 10. Interruption and state safety

- Animations may complete visually after cancellation only if they cannot expose stale or private data.
- On route change, server correction, reconnect, or phase invalidation, cancel or retarget the sequence.
- Keep the authoritative state separate from the visual sequence state.
- Never delay removal of an eliminated player, hidden clue, or private panel until after a cosmetic exit completes.
- Use completion callbacks for focus movement and archive persistence, but do not make business correctness depend solely on animation callbacks.

## 11. Performance

High-frequency animation should use transform and opacity. Use layout animation only when understanding the reflow matters.

Rules:

- Avoid per-frame React state updates.
- Cap particle counts and device pixel ratio.
- Pause offscreen and hidden-tab effects.
- Use `content-visibility` only after verifying accessibility and scroll behavior.
- Avoid large full-screen blurs animated per frame.
- Prefer pre-rendered scene layers and bounded compositor-friendly effects.
- Test on a mid-range mobile profile, not only a desktop browser.

## 12. Cleanup

Clean up:

- GSAP timelines and ScrollTriggers.
- Motion controls and layout groups.
- `requestAnimationFrame` loops.
- Canvas render loops and observers.
- WebGL renderer, geometries, materials, textures, and render targets.
- Event listeners and media-query listeners.
- Audio contexts, nodes, and timers.

React effects that subscribe to external animation or browser state must return cleanup.

## 13. Sound synchronization

- Visual state must complete meaningfully without sound.
- Sound onset may align with the primary beat but cannot be the only cue.
- Avoid loud audio during focus changes or surprise reveals without user setting support.
- Provide a mute/volume path and persist the preference.

## 14. Motion tests

Verify:

- Normal motion.
- Reduced motion.
- Rapid repeated triggering.
- Cancellation and route change.
- Slow network and failed asset.
- Hidden tab and resume.
- Low-performance mobile.
- Keyboard focus during and after animation.
- No stale content flash after authoritative correction.
- No animation queue growth or duplicate timelines.

## 15. Prohibited patterns

- Infinite animation behind body text.
- Animating every card with the same entrance.
- Scroll hijacking for ordinary product pages.
- Layout-shifting hover states.
- Large blur filters animated on low-end devices.
- A cinematic delay before revealing the result the server already returned.
- Private content in a shared transition layer.
- Success animation before server confirmation for game-critical actions.
- New animation dependency without a written capability reason.
