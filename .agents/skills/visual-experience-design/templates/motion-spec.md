# Motion Spec

Use this template for a narrative reveal, cinematic phase transition, complex component transition, or any sequence longer than roughly one second.

## 1. Identity

- Name:
- Product/page:
- Trigger:
- Motion tier:
- Business purpose:
- Narrative purpose:
- Emotional target:

## 2. State contract

Before sequence:

- Authoritative state:
- Visible to user:
- Hidden/private:
- Controls available:

During sequence:

- New information shown per beat:
- Controls locked/unlocked:
- Interruptible:
- What happens on reconnect/route change:

After sequence:

- Authoritative state:
- Persistent visual change:
- Focus destination:
- Controls available:
- Replay behavior:

## 3. Beat timeline

| Beat | Time | Visual change | Information revealed | Easing/motion | Audio/haptic | Callback |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 0ms |  |  |  |  |  |
| 2 |  |  |  |  |  |  |
| 3 |  |  |  |  |  |  |
| 4 |  |  |  |  |  |  |

Total duration:

Cancellation point:

Fallback final frame:

## 4. Technical design

Engine:

Why this engine:

Main animated properties:

Layout/paint risk:

DOM/clone requirements:

Animation instances:

Cleanup:

## 5. Reduced motion

- Static or short transition:
- Information preserved:
- Delays removed:
- Ambient particles:
- Sound:
- Focus flow:

## 6. Performance fallback

Tier A:

Tier B:

Tier C:

Asset failure behavior:

Offscreen/hidden-tab behavior:

## 7. Privacy and authority

- Private content excluded from transition layers:
- No success before server confirmation:
- Stale event handling:
- Reconnect behavior:

## 8. Acceptance

- [ ] Each beat reveals information in a valid order.
- [ ] Essential content is not delayed unnecessarily.
- [ ] Sequence can interrupt and settle safely.
- [ ] Reduced-motion path communicates the same result.
- [ ] Critical private data never appears in shared layers.
- [ ] Animation cleanup is verified.
- [ ] Mobile performance is acceptable.
- [ ] Persistent state follows the reveal.
