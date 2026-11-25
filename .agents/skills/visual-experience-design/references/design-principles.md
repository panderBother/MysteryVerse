# Design Principles

Use this reference to make design decisions that hold up beyond surface styling. It is especially relevant when choosing a direction, reviewing a page holistically, or balancing immersion against operational clarity.

## 1. Start with the product's dramatic question

Every screen should answer a question the user currently has.

- Lobby: "What can I play, and how do I get into a room?"
- Scripted murder mystery: "What phase am I in, what do I know, and what can I do now?"
- Werewolf: "What phase is active, what information is mine, and what action is legal?"
- RPG: "Where am I, what changed, who is present, and what can I try?"
- Evidence reveal: "What did I find, why does it matter, and where is it now?"
- Result/replay: "What happened, why did it happen, and what did I miss?"

If the composition does not make the current question obvious, simplify before adding atmosphere.

## 2. Separate operational clarity from narrative immersion

Immersion is not permission for ambiguity. Use explicit layers:

- Operational UI is predictable, persistent, and readable.
- Narrative UI carries mood, pacing, and interpretation.
- Atmospheric UI creates place and depth.

The layers may visually overlap, but their behavior must not. Timers, legal actions, privacy scope, reconnect state, and error recovery remain clear even during a cinematic moment.

## 3. Design around information asymmetry

MysteryVerse depends on knowing who can see what. Visual design must make visibility a first-class property:

- Public, role-private, faction-private, discoverer-private, and system-only content should have distinct labels and containment.
- Blur or grain can suggest secrecy, but it cannot be the only privacy signal.
- Private content must never appear in shared screenshots, transition ghosts, loading skeletons, toast previews, or animation clones.
- Motion must not reveal the shape or length of hidden content.

## 4. Use narrative state as a layout input

The page shell may stay stable while its internal hierarchy changes by phase:

- Reading phase emphasizes private role material and readiness.
- Investigation phase emphasizes search points, remaining attempts, and emerging evidence.
- Discussion phase emphasizes public clues, speech, and questions.
- Voting phase emphasizes candidates, confirmation, and time.
- Result phase emphasizes causality, outcome, and replay.

Do not force every phase into the same equal-weight grid. Keep a stable shell, then change focal region, density, and action priority.

## 5. Make every visual treatment earn its place

Ask these questions for each major element:

1. Does it communicate state, identity, relationship, hierarchy, affordance, or place?
2. Would the interface be clearer or more emotionally coherent without it?
3. Is it repeated as a meaningful system or used only once for decoration?
4. Does it survive mobile, reduced motion, low contrast, and missing assets?

Remove elements that fail the first question and cannot pass the third.

## 6. Use contrast as composition, not just color

Contrast can come from:

- Scale: one dominant statement against smaller evidence.
- Density: a quiet field around one complex object.
- Material: rough paper against precise metal.
- Temperature: warm evidence against cool environment.
- Motion: a single moving focal element in a still scene.
- Occlusion: foreground framing that directs the eye.

This prevents the common failure where every panel has the same brightness, radius, border, and shadow.

## 7. Design failures as scenes, not alerts

Errors in a game or narrative product still belong to the world:

- A failed clue request can return an empty search-point state with a retry action.
- A stale RPG action can show the scene version changed and offer the nearest safe alternative.
- A disconnected werewolf player can see "reconnecting" around their own seat without exposing hidden actions.
- A failed AI narrator can switch to a templated phase report while preserving rules and controls.

Use plain language for consequences first, thematic styling second.

## 8. Prefer composition over decoration

When a design feels weak, check these before adding effects:

- Is there a clear focal region?
- Is the viewport overcrowded by equal-weight cards?
- Is typography doing enough work?
- Is content grouped by user decision rather than database object?
- Is there enough negative space around the primary action?
- Is the eye led through the scene by light, scale, or alignment?
- Are secondary panels visually subordinate?

Decoration should answer a composition problem, not conceal one.

## 9. Use density according to task

- Discovery/browse surfaces benefit from scannable density and stable card anatomy.
- Gameplay surfaces need a persistent operational frame with dense private context available on demand.
- Narrative moments need controlled pacing and fewer competing signals.
- Creator/editor surfaces need high information density, strong object hierarchy, and predictable panels.
- Result/replay surfaces need chronology and comparison, not decorative spectacle.

Do not apply the same density to a home page, evidence reveal, and rule editor.

## 10. Treat motion as grammar

Motion should express one of these relations:

- **Arrival:** something enters the user's actionable world.
- **Causality:** action produces a result.
- **Continuity:** an object moves between contexts.
- **Relationship:** two pieces of evidence connect.
- **Disclosure:** hidden information becomes available.
- **Transition:** the world or phase changes.
- **Attention:** a time-critical control needs notice.

If the motion expresses none of these, it is probably ambient decoration and must stay quiet.

## 11. Build for interruption and recovery

Real-time interfaces are interrupted by disconnects, stale versions, duplicate events, tab visibility, and user cancellation. Design the interrupted state:

- Can the animation stop without leaving phantom data?
- Can the user retry without repeating a confirmed action?
- Does a reconnect rebuild from authority instead of replaying decorative state?
- Does returning to the tab resume ambient effects responsibly?

## 12. Verify with evidence

A design is not complete because the code compiles. Use rendered output and interaction:

- Screenshot desktop and mobile.
- Exercise the primary flow.
- Check typography, image crop, contrast, overflow, focus states, and layering.
- Check the reduced-motion path.
- Check the browser console and failed assets.
- Compare the result to the intended brief, not just to personal taste.

## Research Anchors

Use these sources as decision references, not as material to copy:

- [Anthropic Agent Skills overview](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview): skills use a required `SKILL.md`, metadata, and progressively loaded resources.
- [Anthropic: Equipping agents with Agent Skills](https://www.anthropic.com/research/equipping-agents-for-the-real-world-with-agent-skills): code and bundled resources can act as reusable capability.
- [shadcn/ui introduction](https://ui.shadcn.com/docs): components are distributed into the product codebase; this is not a substitute for a product design system.
- [Radix Primitives accessibility](https://www.radix-ui.com/primitives/docs/overview/accessibility): accessible behavior, focus management, and keyboard navigation belong in primitives.
- [Motion for React](https://motion.dev/docs/react): React animation, layout motion, presence, and gesture support.
- [Motion accessibility](https://motion.dev/docs/react-accessibility): reduced-motion settings can disable transform and layout motion while preserving other visual changes.
- [GSAP timelines](https://gsap.com/docs/v3/GSAP/Timeline/): timeline sequencing for controlled, multi-beat animation.
- [GSAP ScrollTrigger](https://gsap.com/docs/v3/Plugins/ScrollTrigger/): scroll-linked animation with coordinated refresh and performance behavior.
- [Magic UI components](https://magicui.design/docs/components): a broad catalog of animated visual components and effects.
- [React Bits](https://github.com/DavidHDev/react-bits): creative React text, background, and interaction components.
- [Aceternity UI docs](https://ui.aceternity.com/docs): animated React/Next.js effects and blocks built around Tailwind and Motion.
- [21st.dev](https://21st.dev/): community React component, template, and theme registry with shadcn-compatible installation.
- [Vercel Web Interface Guidelines](https://vercel.com/design/guidelines): interface quality, interaction, accessibility, and performance expectations.
- [WCAG 2.2](https://www.w3.org/TR/wcag/): accessibility success criteria, including animation from interactions.
- [W3C C39 reduced motion](https://www.w3.org/WAI/WCAG22/Techniques/css/C39): use `prefers-reduced-motion` to prevent non-essential interaction motion.
- [web.dev animation performance](https://web.dev/articles/animations-and-performance/): prefer transform and opacity for modern browser animation performance.
- [React useEffect](https://react.dev/reference/react/useEffect): subscriptions and external systems require cleanup.
- [Three.js WebGLRenderer](https://threejs.org/docs/pages/WebGLRenderer): renderer lifecycle, animation loop, and disposal.
- [Three.js object disposal](https://threejs.org/manual/pages/how-to-dispose-of-objects.html): geometry, materials, textures, and renderer resources require explicit cleanup.

Use component libraries selectively:

- **Production foundation candidates:** accessible primitives, project-owned components, Motion for React, CSS, and GSAP when sequencing demands it.
- **Reference catalogs:** Magic UI, React Bits, Aceternity UI, and 21st.dev are strong sources for technique and visual study. Imported components still require accessibility, state, performance, and brand adaptation.
- **Avoid wholesale adoption:** do not add a catalog, animation engine, or 3D runtime merely because it appears in a reference project.
