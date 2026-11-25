# Mystery Scene Example: Rain-Night Study

This is an implementation-ready example for a scripted murder mystery search phase. It demonstrates how place, evidence, operational UI, and reduced-motion behavior work together.

## Scenario

- Mode: AI scripted murder mystery.
- Scene: old harbor manor, second-floor study, 23:40, heavy rain.
- Phase: public investigation, two search attempts remaining.
- Player goal: inspect the room for a discrepancy in the victim's recorded timeline.
- Current question: "What changed in this room, and where should I search?"
- Primary action: choose a search point.
- Privacy: search points are public; resulting evidence can be public, role-private, or discoverer-private.

## Dramatic composition

Focal object: a stopped carriage clock on the desk, face visible at 9:17.

The clock is the primary sightline because:

- It is the sharpest object in the midground.
- A narrow warm desk lamp isolates it from the cool room.
- The room's window, rain, and wet floorboards lead toward the desk.
- The phase action sits below the clock without covering it.

Secondary evidence candidates:

- A chair pulled away from the desk.
- Wet footprints ending at the rug.
- A broken window latch.
- A half-burned letter in the fireplace.
- A portrait with a fresh dust outline.

Each hotspot must map to an actual `SearchPoint` or visible scene fact. No decorative clue object may imply unsupported logic.

## Layer plan

| Layer | Content | Visual role | Interaction |
| --- | --- | --- | --- |
| Ambient | rain haze, distant harbor light | establishes storm and isolation | none |
| Background | bookshelves, tall window, dark corridor | place scale | none |
| Midground | desk, lamp, carriage clock, chair | investigation focus | hotspots |
| Foreground | wet glass, chair back, rain streaks | framing and depth | masked from text/actions |
| Light | warm left lamp, cool right window | directs eye to clock | state-dependent |
| Operational overlay | phase, timer, search count, action | task clarity | fixed |
| Private panel | role card and evidence scope | context | desktop side / mobile tab |

## Desktop layout

Use an edge-to-edge scene rather than placing the room in a rounded card.

```text
+--------------------------------------------------------------+
| phase / timer / room members / settings                      |
+-----------------------------------------------+--------------+
|                                               | role card    |
| scene: study                                  | search count |
| focal: stopped clock                          | evidence     |
| hotspots: desk, window, fireplace, rug        | private chat |
|                                               |              |
| narrative caption                             |              |
+-----------------------------------------------+--------------+
| public discussion / legal phase action                       |
+--------------------------------------------------------------+
```

Rules:

- The scene focal object remains uncovered by the right panel.
- The phase action occupies a stable bottom bar and never floats over a hotspot.
- Right panel width remains stable when evidence count changes.
- Narrative text uses a local dark backplate or mask over detailed wallpaper.

## Mobile layout

The first viewport contains:

- A top phase/timer strip.
- An art-directed scene crop centered on the desk clock and window.
- One short scene caption.
- A fixed bottom primary action.

Persistent tabs:

- 剧情
- 线索
- 聊天
- 角色

The scene hotspot list remains accessible through a labeled "搜证点" button. Tapping a hotspot opens a compact detail sheet. The bottom action is never hidden by the tab bar.

## Scene state contract

```ts
type StudySceneState = {
  sceneId: "harbor-manor-study";
  phase: "investigation";
  weather: "storm";
  time: "23:40";
  searchAttemptsRemaining: number;
  focusedSearchPointId: string | null;
  discoveredSearchPointIds: string[];
  visibleSearchPointIds: string[];
  claimedPrivateSearchPointIds: string[];
  dangerLevel: 0 | 1 | 2;
};
```

Derived visuals:

- `searchAttemptsRemaining === 0`: bottom action changes to "结束搜证并进入讨论" when rules allow.
- `claimedPrivateSearchPointIds`: show "已被发现" or "不可重复获取" without revealing who owns private evidence.
- `dangerLevel`: increases rain/fire intensity only within bounded thresholds; never overlays text.
- Discovered focused point: replaces hotspot pulse with a stamped/archived marker.

## Hotspot anatomy

Each hotspot needs:

- Visible location marker.
- Name on hover/focus.
- Short descriptor.
- Cost/attempt indicator when relevant.
- Public/private outcome scope after acquisition, not before discovery if that would leak.
- Keyboard focusable button or list alternative.

Visual states:

| State | Treatment |
| --- | --- |
| Available | thin amber ring, subtle breathing light at low amplitude |
| Hover/focus | ring sharpens, label appears, scene remains still |
| Selected | locator expands into a compact sheet |
| Pending | locator locks, label "搜查中" |
| Discovered | amber ring becomes metal archive stamp |
| Claimed/closed | muted crossed marker, label "已被发现" |
| Unavailable | no locator; accessed only through disabled list reason if useful |

Do not use strong pulsing rings on every hotspot. Keep one available focal point strongest.

## Lighting and materials

CSS layer variables:

```css
.study-scene {
  --scene-moon: #6f8b9a;
  --scene-fire: #b56c3f;
  --scene-wall: #171b1d;
  --scene-paper: #b9aa91;
  --scene-fog: rgb(126 148 158 / 0.12);
  --scene-backplate: rgb(8 10 12 / 0.78);
}
```

Use:

- Moonlight on the window edge and clock rim.
- Warm lamp pool on desk paper.
- Cool reflection on wet floor.
- Foreground rain only outside the reading/action corridors.

Avoid:

- Purple neon.
- Full-screen blur.
- Equal glow around every object.
- Rain animation over the narrative body text.

## Interaction flow

1. Player focuses a search point by click/touch or keyboard.
2. Local selection expands a detail sheet with name, attempt cost, and search action.
3. Player confirms search.
4. UI shows "搜查中" and locks repeated submission.
5. Server returns the authorized evidence or failure state.
6. The evidence reveal sequence from `evidence-reveal-example.md` runs.
7. Search count, hotspot state, and archive update persistently.
8. Focus returns to the next legal search point or evidence heading.

The visual sequence must start from the confirmed event. Do not show a discoverer-private clue before the server response.

## Scene motion

Entry, normal motion:

1. Scene fades from soft black over 420ms.
2. Warm lamp pool establishes over 280ms.
3. Ambient rain becomes visible at 30% opacity.
4. Phase and action UI settle in the final 220ms.

No large parallax. No full-screen zoom.

Reduced motion:

- Scene appears as a static image immediately.
- Phase and action use opacity-only 120ms transition.
- Rain becomes a static texture.
- Hotspot focus uses border/contrast, not pulsing.

## Performance tiers

### Full

- Layered bitmap scene.
- CSS rain foreground.
- Local lamp glow.
- Low-density dust particles.

### Balanced

- Static layered scene.
- Static rain texture.
- No dust.

### Reduced

- One responsive scene image.
- Reduced local light layers.
- No continuous ambient motion.

## Accessibility

- Scene has an accessible description focused on actionable facts.
- Each hotspot is a button with a meaningful name.
- A "搜索点列表" provides the same choices in document order.
- Phase/timer are announced without reading every second.
- The bottom phase action has a visible focus ring.
- Evidence reveal respects reduced motion.
- Rain and fire ambience have volume controls; all state remains clear with sound off.

## Acceptance

- Desktop focal object is partly visible even with the right panel open.
- Mobile crop still shows the clock and the window-light conflict.
- All search points map to actual rules data.
- Primary action remains visible at 360x800 and 1440x900.
- No private result appears before server confirmation.
- Reduced motion keeps the scene understandable.
- The screen feels like a playable room, not a card containing a background image.
