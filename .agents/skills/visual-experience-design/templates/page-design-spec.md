# Page Design Spec

Use this template before implementing a substantial page, flow, or redesign. Keep it concise enough to guide code and QA; do not turn it into a second PRD.

## 1. Context

- Product:
- Mode:
- Page/route:
- User:
- Current product phase:
- Core task:
- Emotional target:
- Required clarity:
- Privacy scope:
- Existing design system:
- Relevant PRD/source sections:

## 2. Experience brief

One-sentence product promise:

Primary question the page answers:

Primary action:

What must always remain visible:

What must not appear before its legal state:

## 3. Information hierarchy

Operational layer:

- Phase/status:
- Timer:
- Primary action:
- Secondary actions:
- Connection/error:

Interpretive layer:

- Narrative:
- Evidence/clues:
- Role/identity:
- Relationship/team state:

Atmospheric layer:

- Place/scene:
- Light/weather:
- Ambient motion:
- Audio:

## 4. Composition

Desktop focal point:

Desktop regions:

- Header/shell:
- Main:
- Left/side:
- Right/side:
- Bottom:

Mobile focal point:

Mobile view transformation:

- Tabs:
- Drawers:
- Fixed primary action:
- Deferred content:

## 5. Visual language

Theme mode:

Primitive tokens to reuse:

New semantic tokens:

Typography:

- Display:
- Heading:
- Narrative:
- UI:
- Numeric/data:

Color roles:

- Canvas/surface:
- Primary text:
- Secondary text:
- Action:
- Evidence:
- Public:
- Private:
- Danger:
- Success:

Material cues:

Imagery/assets:

## 6. Components

Reused components:

New components:

Card/panel policy:

Icon set:

Evidence/dossier anatomy, if applicable:

## 7. Interaction map

| Action | Legal state | Immediate feedback | Authority result | Failure/recovery |
| --- | --- | --- | --- | --- |
|  |  |  |  |  |

Confirmation requirements:

Keyboard path:

Touch path:

Private-data safeguards:

## 8. Required states

- Initial/loading:
- Empty:
- Partial/streaming:
- Pending:
- Success:
- Failure:
- Reconnect:
- Stale/conflict:
- Permission denied:
- Unavailable mode/action:

## 9. Cinematic sequence

Only complete when the page contains a narrative reveal or phase transition.

- Scene subject:
- Dramatic question:
- Background:
- Midground:
- Foreground:
- Key light:
- Focal object:
- Scene state inputs:
- Transition-in:
- Transition-out:
- Reduced-motion variant:

## 10. Motion summary

| Element | Tier | Purpose | Duration/easing | Reduced motion |
| --- | --- | --- | --- | --- |
|  |  |  |  |  |

Engine choice:

Heavy effects/fallback:

## 11. Responsive behavior

| Breakpoint | Composition change | Primary action | Hidden/deferred UI | Asset crop |
| --- | --- | --- | --- | --- |
| Compact |  |  |  |  |
| Medium |  |  |  |  |
| Wide |  |  |  |  |
| Ultrawide |  |  |  |  |

## 12. Accessibility

- Landmark/heading structure:
- Focus order:
- Dialog/drawer semantics:
- Icon labels:
- Contrast checks:
- Non-color state cues:
- Reduced motion:
- Sound/flash/shake controls:
- Spatial/keyboard alternative:

## 13. Implementation plan

Files:

Dependencies:

Data/contracts:

Asset needs:

Risks:

Verification plan:

## 14. Acceptance criteria

- [ ] Primary task is obvious and reachable on desktop and mobile.
- [ ] Current phase and legal action remain visible.
- [ ] Private information is not present in public layers.
- [ ] All operational states are designed.
- [ ] Motion has purpose, interruption behavior, and reduced-motion path.
- [ ] Visual system uses semantic tokens and existing primitives.
- [ ] Browser screenshots and interaction QA pass.
