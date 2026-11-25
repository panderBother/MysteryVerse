# Interaction Patterns

Use this reference when designing controls, forms, overlays, navigation, feedback, real-time state, or game actions. Every visible control needs a legal state, feedback path, keyboard path, mobile equivalent, and recovery path.

## 1. Interaction model

Classify each interaction:

- **Navigation:** changes page, view, tab, stage, or scene.
- **Selection:** chooses an object, role target, clue, seat, or filter.
- **Input:** accepts text, file, number, or option.
- **Commitment:** creates a durable or game-critical result.
- **Disclosure:** reveals additional detail or private context.
- **Comparison:** relates, pins, filters, or contrasts information.
- **Recovery:** reconnects, retries, rolls back, or resolves a conflict.

Commitment actions need stronger confirmation and authority handling than navigation and disclosure.

## 2. Visual affordance

- Make clickable surfaces look actionable without making all content look like a button.
- Use hover and focus to preview action, not to conceal essential labels.
- Disabled controls should preserve layout and expose a reason when the rule is non-obvious.
- Destructive actions must be visually distinct and use explicit verbs.
- Selection state needs shape, border, label, or indicator beyond a subtle color shift.
- Draggable objects need a handle or clear drag region and a non-drag fallback.

## 3. Buttons and command controls

Every button contract:

| State | Behavior |
| --- | --- |
| Default | stable label and accessible name |
| Hover | subtle lift/brightness; no layout shift |
| Focus-visible | high-contrast ring |
| Pressed | immediate inset/scale/opacity response |
| Loading | stable size, spinner or progress, disabled repeat submission |
| Disabled | reduced emphasis, reason available |
| Error | returns to usable state with nearby explanation |
| Success | brief confirmation when needed; persistent state elsewhere |

Use icon+text for unfamiliar tools. Use icon-only controls only for familiar symbols, with tooltip and accessible label.

## 4. Forms

- Label every field visibly.
- Use one primary submission action per form region.
- Group fields by user decision or story structure.
- Preserve values after recoverable error.
- Validate at a useful moment; avoid punishing incomplete typing.
- Explain business constraints in context.
- File upload must show accepted types, size, progress, parsing state, failure, retry, and cancellation.
- Long creator forms need draft status and safe navigation away.

### Free-form RPG action input

- Keep the scene and current character context visible.
- Accept natural language but show how the system interprets it before committing when the action changes world state.
- Offer legal example actions or recent actions, not a hidden script.
- Distinguish "understanding", "needs clarification", "invalid", "pending", "settled", and "narrated".
- Never generate narration for an action that failed rule validation.

## 5. Dialogs

Use for:

- Confirmation with meaningful consequence.
- A focused decision that benefits from blocking context.
- Short critical information.

Do not use for long reading, complex editing, or routine navigation.

Requirements:

- Modal semantics.
- Accessible title.
- Initial focus on a useful control.
- Focus trap while open.
- Escape/close path unless the business rule requires acknowledgment.
- Focus returns to invoker.
- Background interaction disabled.
- Consequence, affected object, and irreversible scope stated.

For cinematic reveals, the visual layer may extend beyond the dialog, but control semantics still need a clear interactive surface.

## 6. Sheets and drawers

Use for side context on mobile or secondary tools on desktop.

- State what the drawer contains in the trigger and title.
- Preserve the main scene context where possible.
- Support keyboard close and focus return.
- Avoid stacking multiple drawers with ambiguous escape order.
- Do not hide the current phase's primary action behind a drawer.

## 7. Tabs and segmented controls

- Use tabs for peer views within one context.
- Use segmented controls for compact mutually exclusive modes.
- Use a menu for many options or long labels.
- Keep labels visible; do not depend on icons alone.
- Animate the selection indicator briefly.
- If panels change height, keep the primary action location predictable.

## 8. Tooltips and popovers

Tooltips explain unfamiliar icons or compact metadata. They do not replace labels for form fields.

- Appear on hover and focus.
- Dismiss on escape.
- Do not contain primary actions.
- Keep within viewport.
- Do not expose private information in a shared/public context.

Popovers may contain controls. They need focus management, collision handling, and a clear close behavior.

## 9. Menus and selects

- A visible selected value is required for a select.
- Menu options need readable labels and keyboard navigation.
- Group large option sets.
- Avoid custom select menus when a native select is sufficient.
- Selection changes must not silently commit a dangerous game action.

## 10. Loading and progress

Distinguish:

- Instant local acknowledgement.
- Pending server request.
- Long-running queued job.
- Streaming generation.
- Authoritative confirmation.
- Retryable failure.
- Non-retryable failure.

### AI generation pipeline

Show:

- Current stage name.
- Completed stages.
- Active stage.
- Remaining stages or estimated behavior.
- "Still processing" after a threshold.
- Failure reason and safe retry.
- Partial artifacts only when safe to display.

Do not use one indefinite spinner for a multi-stage task.

### Skeletons

- Match final layout.
- Cap shimmer intensity.
- Do not shimmer entire pages when only one panel is loading.
- Replace with real content without large reflow.
- Respect reduced motion.

## 11. Empty, error, and recovery states

### Empty

An empty state should:

- Explain why the area is empty.
- Show the next legal action.
- Avoid humorous or decorative copy when the user may be confused.
- Preserve the current game context.

### Error

- State what happened.
- State what did not happen.
- Provide the next safe action.
- Include a stable error/reference ID for support when available.
- Do not expose stack traces, private clues, or AI reasoning.

### Reconnect

- Keep the last confirmed state visible.
- Mark the connection state globally and locally.
- Block duplicate or stale submissions.
- Show which current game action is unavailable.
- On reconnect, rebuild from snapshot plus missing events.
- Remove the reconnect UI only after authority is synced.

### Conflict/stale state

- Explain that the world or room changed.
- Show the nearest legal alternative.
- Do not replay a conflicting animation as if it succeeded.
- Preserve user input when possible.

## 12. Navigation

- Keep current mode, room, phase, and privacy context visible.
- Use stable labels, not only thematic icons.
- Avoid hiding primary navigation behind multiple unlabeled icon layers.
- Page transitions should preserve orientation through shared title, scene, or stage.
- Back behavior must be explicit in a game flow. If leaving would forfeit or disconnect, warn with consequence.

## 13. Toast and transient feedback

Use toast for non-blocking, non-critical confirmation:

- Copied link.
- Saved draft.
- Invitation sent.

Do not use toast alone for:

- Vote result.
- Clue acquisition.
- Role reveal.
- Elimination.
- Purchase/transaction confirmation.
- Error requiring recovery.

Critical confirmation should leave a persistent UI change and, when appropriate, an event/timeline entry.

## 14. Game-specific interactions

### Scripted murder mystery

- Search point selection: show location, cost, availability, and whether others can observe.
- Evidence viewing: public/private scope is explicit.
- Evidence comparison: preserve selected pair and allow keyboard/list alternative to spatial board.
- Questioning: separate public question, private question, and DM clarification.
- Voting: candidate list, current vote, abstention rule, confirmation, server result.
- DM hint: show hint level, known cost/limit, and that it does not reveal unsupported facts.

### Werewolf

- Night action: role-private region, legal target list, pending confirmation, no public target preview.
- Speaking: queue position, remaining time, microphone/text availability.
- Vote: current selection, locked/confirmed state, public tally only when rules permit.
- Elimination: persist death state without exposing unrevealed role.
- Phase transition: update day/night controls and legal actions before ambient choreography completes.

### RPG

- Movement: destination, reachability, travel cost, party confirmation.
- Scene interaction: object/NPC hotspot and a text/list alternative.
- Dialogue: visible scope (public/private/queue), NPC presence, context, and world-action confirmation.
- Inventory/task: no blank card when empty; show exploration guidance.
- Multiplayer world: separate public and personal event streams.

## 15. Keyboard and focus

- All primary flows must be keyboard accessible.
- Focus order follows reading and action order.
- Focus-visible remains visible over scene imagery.
- Hotspots need a tab-accessible equivalent.
- Drag, canvas, and spatial boards need list or keyboard controls for equivalent operations.
- Do not trap focus in non-modal panels.
- Preserve focus when items reorder unless moving focus is the explicit user request.

## 16. Touch and pointer

- Minimum practical hit area: 44x44px.
- Do not place independent small targets too close.
- Avoid hover-only information on mobile.
- Support tap feedback.
- Ensure scrollable game regions do not conflict with drag gestures.
- Respect safe-area insets for bottom actions and sheets.

## 17. Sound and settings

Provide controls for:

- Master volume.
- Ambience.
- Effects.
- Reduced motion.
- Screen shake/flashing.
- Haptics where supported.

Persist preferences and apply before the next scene transition. An accessibility setting cannot require the user to repeat the cinematic sequence to take effect.

## 18. Interaction acceptance

- The current primary action is visible and reachable.
- All legal actions have a pending/confirmed/failed path.
- Illegal or stale actions explain recovery.
- Private data has no public preview.
- Keyboard, touch, reduced motion, and reconnect flows work.
- Feedback persists where the consequence persists.
- Loading does not resize stable controls or hide current state.
- No interaction depends solely on hover, color, sound, or animation.
