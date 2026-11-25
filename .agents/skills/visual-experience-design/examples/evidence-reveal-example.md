# Evidence Reveal Example: Key Clue Acquisition

This example defines the complete visual process for discovering a game-critical clue in MysteryVerse. The reveal is driven by an authoritative evidence event, uses the archive metaphor, and ends with a persistent product state change.

## Scenario

- Mode: AI scripted murder mystery.
- Event: the player searches the study fireplace.
- Result: discoverer-private evidence `letter_burn_fragment`.
- Evidence title: "未烧尽的航运收据".
- Key fact: the victim's claimed 21:00 departure is contradicted by a dock receipt timestamped 21:26.
- Consequence: add the evidence to the player's archive, update private clue count, and create a timeline node.
- Privacy: only the discoverer sees the full reveal.

## Narrative goal

The player should experience:

1. "I found something."
2. "This object is evidence, not set dressing."
3. "Here is the exact fact I can use."
4. "It is now part of my case file."

The sequence cannot merely show a modal with a title. It must transform the clue from scene object to readable evidence to persistent archive artifact.

## Visual anatomy

Evidence card:

- Source strip: "书房 · 壁炉".
- Visibility seal: "发现者可见".
- Evidence title.
- Image/excerpt.
- Readable fact callout.
- Keywords: `21:26`, `码头收据`, `离港时间`.
- Context line: "与某角色的公开时间线存在冲突".
- Archive action/status.

No unrelated generic buttons. Primary action after reveal: "收入档案" if the rules require explicit archiving; otherwise show "已收入档案" and expose "查看线索".

## Timeline: normal motion

| Beat | Time | Motion | Information state |
| --- | --- | --- | --- |
| 1. Anchor | 0-180ms | scene dims locally around fireplace; operational controls remain visible | source location remembered |
| 2. Emerge | 120-420ms | evidence card rises 18px and scales 0.96 to 1; backdrop opacity 0.15 to 0.72 | card shape and source visible |
| 3. Flip | 380-760ms | card rotates on Y with a soft edge shadow; back face contains sealed evidence | title and scope become readable |
| 4. Read | 700-1030ms | title, excerpt, fact callout, and keywords stagger by 45ms | evidence can be understood |
| 5. Connect | 960-1160ms | one amber thread grows from the evidence to a compact timeline target | contradiction becomes spatial |
| 6. Archive | 1080-1380ms | card moves/settles toward the clue archive; archive count animates to new confirmed value | persistent ownership becomes visible |
| 7. Return | 1380-1500ms | overlay recedes; controls unlock; focus lands on evidence heading or archive item | new state is usable |

Total: approximately 1.5s.

Do not add particles, full-screen blur, or camera shake to this sequence. The evidence itself is the spectacle.

## Layout

Desktop:

- Reveal layer is centered over the gameplay scene.
- Evidence card width: 480-560px.
- Scene source hotspot remains faintly visible behind the card so origin is preserved.
- Clue archive count sits at the top-right of the evidence panel, outside the card.
- Timeline target appears as a small inset or edge marker, not a second full panel.

Mobile:

- Use a near-full-height evidence sheet with safe-area bottom padding.
- Flip remains vertical-axis but with reduced travel.
- Timeline connection becomes a short horizontal relation chip below the fact.
- Archive action is fixed at the bottom of the sheet.
- Scene is visible as dim context at the top edge, then recedes.

Reduced motion:

- No card flip or spatial travel.
- Evidence sheet fades in at 120ms.
- Content is immediately readable in final order.
- Archive count and timeline marker update with a 120ms highlight.
- Focus moves to evidence heading.

## Interaction contract

Trigger:

- Authoritative `EvidenceDiscovered` event addressed to this player.

Preconditions:

- Evidence visibility grants this player access.
- The event is not a duplicate replay.
- Route still matches the room and scene.

During:

- User can press Escape or "跳过动效" after beat 3.
- Skipping goes immediately to the final readable state.
- Focus is inside the reveal surface until dismissal.
- Background game actions are disabled while the modal reveal is open.

After:

- Evidence exists in the archive.
- Private clue count reflects server state.
- Timeline node exists if the rule produces one.
- Focus returns to source hotspot, next legal search point, or archive heading.

Replay:

- Does not automatically replay when revisiting the archive.
- User may request "重看发现过程" from evidence detail if product policy allows.

## Motion implementation sketch

This sketch shows ownership and reduced-motion handling. Adapt class names and state store to the repository.

```tsx
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { useEffect, useRef, useState } from "react";

type EvidenceRevealProps = {
  evidence: {
    id: string;
    title: string;
    sourceLabel: string;
    fact: string;
    keywords: string[];
    scopeLabel: string;
  } | null;
  onDismiss: () => void;
};

export function EvidenceReveal({ evidence, onDismiss }: EvidenceRevealProps) {
  const reduceMotion = useReducedMotion();
  const headingRef = useRef<HTMLHeadingElement>(null);
  const [phase, setPhase] = useState<"emerge" | "flip" | "read" | "archive">(
    "emerge",
  );

  useEffect(() => {
    if (!evidence) return;
    setPhase("emerge");
    const flipTimer = window.setTimeout(
      () => setPhase("flip"),
      reduceMotion ? 0 : 260,
    );
    const readTimer = window.setTimeout(
      () => setPhase("read"),
      reduceMotion ? 0 : 650,
    );
    const archiveTimer = window.setTimeout(
      () => setPhase("archive"),
      reduceMotion ? 0 : 980,
    );
    headingRef.current?.focus();

    return () => {
      window.clearTimeout(flipTimer);
      window.clearTimeout(readTimer);
      window.clearTimeout(archiveTimer);
    };
  }, [evidence, reduceMotion]);

  const cardTransition = reduceMotion
    ? { duration: 0.12 }
    : { duration: 0.32, ease: [0.16, 1, 0.3, 1] as const };

  return (
    <AnimatePresence>
      {evidence ? (
        <motion.div
          className="evidence-reveal"
          role="dialog"
          aria-modal="true"
          aria-labelledby="evidence-reveal-title"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: reduceMotion ? 0.1 : 0.2 }}
        >
          <motion.article
            className="evidence-reveal__card"
            initial={reduceMotion ? { opacity: 0 } : { opacity: 0, y: 18, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            transition={cardTransition}
            data-phase={phase}
          >
            <header className="evidence-reveal__source">
              <span>{evidence.sourceLabel}</span>
              <strong>{evidence.scopeLabel}</strong>
            </header>

            <h2
              id="evidence-reveal-title"
              ref={headingRef}
              tabIndex={-1}
            >
              {evidence.title}
            </h2>

            <p className="evidence-reveal__fact">{evidence.fact}</p>

            <ul aria-label="关键证据词">
              {evidence.keywords.map((keyword, index) => (
                <motion.li
                  key={keyword}
                  initial={reduceMotion ? false : { opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{
                    duration: reduceMotion ? 0 : 0.24,
                    delay: reduceMotion ? 0 : index * 0.045,
                  }}
                >
                  {keyword}
                </motion.li>
              ))}
            </ul>

            <footer className="evidence-reveal__actions">
              <button type="button">查看线索</button>
              <button type="button" onClick={onDismiss}>返回现场</button>
            </footer>
          </motion.article>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}
```

Implementation notes:

- Replace timer-driven visual phases with a GSAP timeline only if the sequence becomes more complex.
- Keep evidence persistence outside this component.
- Do not use the component's timeout callback as the source of truth for clue ownership or counts.
- Add the skip control and trap focus using the project's accessible dialog primitive.
- Use a CSS 3D back face only after testing overflow and Safari behavior.

## Archive motion

The archive transition should preserve continuity:

1. Card edge or seal detaches.
2. A small evidence token travels to the archive trigger.
3. Archive count updates from the authoritative new count.
4. The new evidence row appears with a brief amber settling border.
5. The token dissolves.

If shared-element motion is unstable, use a short path line and row highlight. Do not fake a moving copy of private evidence across the shared room canvas.

## Visual quality rules

- The excerpt is the dominant content, not a decorative icon.
- The visibility seal is readable.
- Keywords look like annotations, not generic chips.
- The card borrows evidence/archive material language; it does not become a glass modal.
- Background scene remains recognizable but subordinate.
- No unrelated secondary actions.
- The final archive row remains findable after the overlay closes.
- Sound cue is optional, short, and never the only confirmation.

## Error and interruption

- If the evidence asset fails: show a designed paper fallback with title, fact, and source; no broken image icon.
- If server state is corrected: cancel archive animation and rebuild from corrected state.
- If disconnect occurs: keep the already-confirmed evidence visible; do not re-submit the search.
- If the player leaves the route: unmount immediately; do not carry private evidence into a route-transition layer.
- If reveal is skipped: preserve all information and persistent state.

## Acceptance

- Trigger begins only after authorized evidence event.
- Private evidence never appears for another player.
- Player can understand the fact without sound or motion.
- Reveal is under 1.6s before final interaction, excluding optional reading.
- Reduced-motion path requires no flip or spatial travel.
- Archive state and count persist.
- Focus moves logically and returns safely.
- The result feels like handling evidence, not opening a generic modal.
