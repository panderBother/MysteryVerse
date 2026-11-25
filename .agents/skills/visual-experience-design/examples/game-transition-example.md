# Game Transition Example: Werewolf Night to Dawn

This example defines a cinematic but rule-safe transition from werewolf night action to public dawn. It changes the environment, the operational phase, and the public information model without exposing hidden roles or actions.

## Scenario

- Mode: AI werewolf.
- Transition: night settlement complete, public dawn begins.
- Players: 6 human, 3 AI.
- Current phase: night action submission and settlement complete.
- Next phase: dawn announcements, death status, last words, speaking queue.
- Privacy rule: night actions remain hidden unless the current rules explicitly reveal them.
- Primary next action: review dawn result and proceed to discussion.

## Narrative goal

The sequence should communicate:

1. Night actions are closed and authoritative.
2. The world has advanced to dawn.
3. Public consequences are being revealed.
4. The room is ready for public discussion.

It should not imply:

- Who acted.
- What failed or succeeded privately.
- Who holds which unrevealed role.
- That AI reasoning has been exposed.

## Scene layers

| Layer | Night state | Dawn state | Transition |
| --- | --- | --- | --- |
| Ambient | deep blue-grey mist | pale warm-grey haze | color/temperature shift |
| Background | moonlit village/ritual square | same geometry, softer horizon | crossfade a pre-authored dawn plate |
| Midground | seat ring under moon silver | seat ring becomes primary readable object | lower vignette, increase seat contrast |
| Foreground | drifting mist | thin residual mist | fade and drift out |
| Light | moon rim from upper right | dawn wash from horizon | directional crossfade |
| Operational header | 夜晚 · 第 2 轮 | 天亮 · 第 2 轮 | immediate semantic update after authority |
| Side panel | private action/settlement state | public dawn result | seal and replace with authorized content |

Do not reuse a role-private transition layer on the public scene.

## Desktop sequence: full motion

| Beat | Time | Visual action | State behavior |
| --- | --- | --- | --- |
| 1. Close night | 0-180ms | private action panel seals; public chrome remains visible | input locked, "等待天亮" |
| 2. World shift | 120-820ms | moon layer fades, dawn plate emerges, mist drifts toward horizon | no new private data |
| 3. Seat reveal | 700-1280ms | public seat states resolve one by one in server order | death/alive state only |
| 4. Phase declare | 1140-1500ms | header transitions to "天亮 · 第 2 轮"; dawn light settles | authoritative phase visible |
| 5. Public result | 1320-1900ms | dawn deaths/last words/announcement panel enters | authorized public events |
| 6. Discussion ready | 1800-2400ms | speaking queue and legal action appear | controls unlock |

Total: approximately 2.4s.

The authoritative phase must be visible by 1.5s at the latest. Do not make discussion controls wait for residual fog animation.

## Mobile sequence

- Preserve the seat ring as a compact circular/grid summary.
- Use a 16:9 or 4:3 dawn scene band above the seat summary.
- Private side panel becomes a bottom sheet that first seals, then closes.
- Dawn result appears as a public sheet with death status and next action.
- Discussion queue is visible after the result sheet is readable.
- The primary "进入讨论" action is fixed and cannot be obscured by ambient effects.

Reduced motion:

- Cut from night to dawn.
- Header changes immediately.
- Public result appears with a 120ms opacity transition.
- Seat states change without stagger.
- No mist travel, parallax, or full-screen crossfade.

## Information choreography

### Private panel sealing

The private action panel should:

1. Stop accepting input.
2. Show a seal mark and short text "夜晚行动已提交".
3. Fade its private target details before the public layer expands.
4. Unmount before any shared dawn layer is cloned or captured.

Do not animate the actual role name, target, or action icon across the center of the room.

### Dawn result

Reveal only event types authorized for public view:

- Death/elimination status.
- Last words availability.
- Hunter shot or other public chain outcome when rules make it public.
- Public announcement text.
- Updated speaking order.

Animate deaths by a state change on the seat, not a decorative generic skull overlay that hides the player's name.

### Speaking queue

- Current speaker is the strongest seat highlight.
- Queue is visible as ordered text, not a carousel.
- Time and turn state use stable widths.
- AI player speaking state is restrained and says "思考中" without internal reasoning.

## State contract

```ts
type DawnTransitionEvent = {
  type: "phase_changed";
  roomId: string;
  from: "night";
  to: "dawn";
  round: number;
  publicEvents: Array<
    | { type: "player_died"; playerId: string }
    | { type: "last_words_available"; playerId: string }
    | { type: "public_announcement"; text: string }
    | { type: "speaking_queue_updated"; playerIds: string[] }
  >;
  eventSeq: number;
};
```

Transition component rules:

- Render only from this authoritative event or a rebuild of the same authoritative state.
- Ignore duplicate `eventSeq`.
- Cancel if route changes.
- Rebuild from snapshot on reconnect instead of replaying an old transition with stale players.

## Engine choice

Use CSS for:

- Header color/contrast state.
- Dawn background crossfade.
- Seat state transitions.
- Result panel opacity/translation.

Use Motion or GSAP only if:

- Seat reveals require precise orchestration with public events.
- There are chained public outcomes such as hunter shot.
- The transition must coordinate several scene layers with a single cancellation point.

Do not use WebGL for this transition unless the werewolf environment is already a real 3D scene with a product reason.

## CSS layer sketch

```css
.werewolf-world {
  position: relative;
  isolation: isolate;
  overflow: hidden;
}

.werewolf-world__night,
.werewolf-world__dawn {
  position: absolute;
  inset: 0;
  background-position: center;
  background-size: cover;
  transition: opacity 700ms var(--ease-standard);
  pointer-events: none;
}

.werewolf-world[data-phase="dawn"] .werewolf-world__night {
  opacity: 0;
}

.werewolf-world[data-phase="dawn"] .werewolf-world__dawn {
  opacity: 1;
}

.werewolf-seat[data-status="dead"] {
  border-color: var(--color-border-strong);
  filter: saturate(0.35);
}

.werewolf-seat[data-status="dead"] .werewolf-seat__status::after {
  content: "已出局";
}

@media (prefers-reduced-motion: reduce) {
  .werewolf-world__night,
  .werewolf-world__dawn {
    transition-duration: 120ms;
  }
}
```

For reduced motion, ensure the dawn plate is immediate even if the opacity rule remains.

## Sound

Optional sound roles:

- Night ambience fades.
- One restrained transition tone.
- Soft public reveal cue when the dawn result becomes readable.
- Discussion ambience begins after controls unlock.

All state remains complete with sound off. Provide ambience/effects controls.

## Visual quality rules

- Dawn is not simply the night image with a yellow overlay.
- The seat ring remains the social focus.
- The transition does not reveal unrevealed roles.
- Public deaths remain readable by name and status.
- The current action is visible before atmospheric motion ends.
- The dawn palette is pale and restrained, not a second generic gradient theme.
- AI speaking state is a small status, not a spectacle that competes with the player whose turn it is.

## Failure and recovery

- Missing dawn asset: use a designed dawn color/paper composition with same phase clarity.
- Event delay: show "等待天亮结算" while keeping the last confirmed night state.
- Duplicate phase event: no second transition.
- Reconnect during transition: load current phase, seat states, and speaking queue; skip old cinematic replay.
- Reduced motion toggled mid-transition: settle to final dawn state immediately.
- Audio failure: continue silently.

## Acceptance

- Night-private target/role information never enters public transition layers.
- Public phase is readable within 1.5s.
- Discuss action is available only after authoritative dawn state.
- Each public outcome maps to an event.
- Death status and speaking queue persist after the cinematic.
- Reduced-motion path communicates all public results.
- Mobile layout keeps seat status and primary action visible.
- No generic skull/fog overlay replaces actual player names and rule outcomes.
