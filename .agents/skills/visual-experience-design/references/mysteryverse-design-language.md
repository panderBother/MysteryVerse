# MysteryVerse Design Language

Use this reference for any MysteryVerse page, component, scene, motion sequence, or visual review. It synthesizes the product direction from `docs/MysteryVerse-PRD.md`: an AI-native multiplayer deduction platform with three modes sharing identity, rooms, content, real-time state, and replay infrastructure.

If an implemented design system exists, map these directions onto its tokens and components rather than replacing it.

## 1. Brand idea

MysteryVerse should feel like entering an active case file, ritual table, or living story world, not like opening a generic AI dashboard.

Core visual metaphor:

- **Archive:** evidence, dossiers, recorded testimony, versioned truth.
- **Threshold:** entering a room, scene, role, phase, or world.
- **Thread:** relationships, contradictions, inference, and consequence.
- **Seal:** private information, role identity, phase authority, final result.

Use these as systems, not literal decoration everywhere:

- Paper and ink for human testimony.
- Metal and stamped labels for system/rule authority.
- Thread, pin, and fold for relationships.
- Light and weather for place and phase.
- Tabular numbering and timestamps for replayable causality.

## 2. Shared brand shell

Across modes, maintain:

- Clear product identity in navigation, not oversized logo treatment.
- Consistent focus, form, overlay, and icon behavior.
- Consistent semantics for public, private, faction, discovery, danger, success, and system data.
- Consistent typography roles.
- Consistent route/page transition orientation.
- Consistent room and phase status language.

Do not force modes into the same center column and right sidebar. Share primitives and logic, not identical composition.

## 3. Common information priorities

Every game page must keep these available:

1. Current mode and room.
2. Current phase and time state.
3. Current legal primary action.
4. Own role/identity context.
5. Privacy scope of visible content.
6. Team/player presence and connection state.
7. Entry to public/private communication permitted by the phase.
8. Error/reconnect recovery.

The visual treatment may change by phase, but the information cannot disappear without a legal reason.

## 4. Privacy language

Use multiple signals:

| Scope | Visual treatment | Persistent label |
| --- | --- | --- |
| Public | open surface, shared thread, normal contrast | "公开" |
| Role-private | sealed/inset surface, warm lock mark | "仅你可见" |
| Faction-private | separate channel color and boundary | faction label + "阵营可见" |
| Discoverer-private | limited evidence tab and owner marker | "发现者可见" |
| System | metal/utility treatment, no narrative frame | "系统" |
| Unrevealed | absence or explicitly sealed state | do not tease hidden content |

Additional rules:

- Do not show blurred private text on a public surface.
- Do not reveal word count, title shape, portrait, or metadata of private content in transitions.
- Use lock/eye/scope icons only alongside text.
- Private screenshots must not include another player's private context in the background.

## 5. Motion identity

Use four motion verbs:

- **Inspect:** a small focus shift, mask, or zoom around evidence.
- **Unseal:** a card/folder edge opens and private information becomes readable.
- **Thread:** a line or relation grows between two evidence nodes.
- **Advance:** a light/time/phase change moves the room into a new state.

Avoid generic "everything fades up". Use motion to model archival handling, ritual phase, or physical movement.

Quiet UI remains 80-220ms. Evidence/role reveals can reach 500-1000ms. Whole-room phase transitions can reach 1.2-3.0s. All need reduced-motion alternatives.

## 6. Mode 1: AI scripted murder mystery

### Visual keywords

Archive, evidence, investigation, secrecy, deduction, testimony, contradiction.

### Primary visual language

- Ink/near-black game shell for active investigation.
- Paper/dossier surfaces for testimony and evidence.
- Warm amber for acquired or actionable evidence.
- Muted green for supported connection or confirmed public fact.
- Red-brown for danger, accusation, contradiction, and irreversible confirmation.
- Metal and stamping for phase authority and system messages.

### Desktop composition

Recommended shell:

- Top: phase, timer, room members, settings.
- Left/main: narrative and current phase action.
- Right: role card, private script, evidence/timeline, private chat entry.
- Bottom: public/private communication area according to phase.

The exact PRD page direction is stage/timer/members at top; story/action in the main region; role card/clues/private-chat entry at right; chat input at bottom.

### Mobile composition

Four persistent views:

- Story.
- Evidence.
- Chat.
- Role.

Keep the current phase primary action fixed and visible. Convert private/side panels to full-height views or tabs rather than narrow columns.

### Search-point design

- Show place, phase availability, remaining attempts, visibility, and expected interaction.
- Use scene hotspots plus an accessible list.
- Selecting a search point must not reveal result before server confirmation.
- If another player claims the last exclusive clue, show a truthful "already discovered" state.

### Evidence card anatomy

- Evidence name.
- Source and acquisition context.
- Visibility scope.
- Image/excerpt.
- Annotations or related facts.
- Actions: inspect, compare, pin, ask DM, archive.
- Timestamp and role/author provenance when permitted.

Do not make every evidence card identical. Testimony, physical evidence, timeline event, and contradiction need distinct anatomy while sharing metadata rules.

### Evidence board

- Spatial board uses meaningful grouping, not random masonry.
- Use pins/threads only for actual relationships.
- Distinguish fact, hypothesis, and contradiction by line style, label, and shape.
- Provide a chronological/list alternative for keyboard and mobile.
- Do not expose others' private evidence through shared positions or missing-node silhouettes.

### Scripted reveal

Final truth reconstruction should:

1. Replay confirmed events in order.
2. Connect evidence to claims.
3. Highlight where the player's theory diverged.
4. Persist the resulting timeline/archive.
5. Avoid revealing excluded private role material that is not part of the permitted result view.

## 7. Mode 2: AI werewolf

### Visual keywords

Ritual, faction, secrecy, pressure, night, judgment, collective decision.

### Primary visual language

- Moon silver as public structural material.
- Oxblood used only for hidden faction or serious danger.
- Night scene uses cool low light; day uses neutral high clarity.
- Seat ring is the dominant social object; don't reduce it to a generic player card grid.
- Speech and vote use precise time/state treatment, not decorative cards everywhere.

### Desktop composition

- Top: day/night, round, timer.
- Center: ring or grid of seats.
- Side panel: role-private night action, identity card, speaking queue, or vote panel depending on phase.
- Bottom: public speech or quick input.

The PRD direction specifies a center seat layout with phase-dependent side panels and mobile bottom drawers.

### Seat anatomy

- Avatar/name.
- Alive/dead/unknown public state.
- Speaking/current queue.
- Vote state when public rules permit.
- Connection state.
- Role/faction only in the authorized private view.

Do not encode unknown roles through obvious border colors that leak data.

### Night

- Public scene reduces brightness and competing motion.
- Private action panel becomes the primary focus.
- Legal target selection is explicit.
- Submission shows "submitted, awaiting settlement", not a fabricated result.
- AI thinking motion is restrained and does not reveal internal reasoning.

### Day

- Clarity increases.
- Speaking queue becomes dominant.
- Public history and vote controls remain accessible.
- Phase change must not obscure current legal action after authority updates.

### Vote and elimination

- Candidate selection, own vote state, lock, deadline, and tally rules are explicit.
- Vote reveal animates from actual events.
- Elimination persists death status and permitted last words.
- Role reveal uses a separate authorized beat; never infer a hidden role from elimination animation.

## 8. Mode 3: Immersive RPG

### Visual keywords

Place, exploration, consequence, companionship, unknown world, living environment.

### Primary visual language

- Environment-led composition, not a chat app overlaying a wallpaper.
- Moss, copper, mineral, weather, and local scene light provide variation.
- Public world events and personal narrative are visually distinct streams.
- NPC portraits/scene subjects reveal actual character and place.
- System/action confirmations use precise metal/utility styling against the organic world.

### Desktop composition

- Left: map/scene navigation.
- Center: scene narrative and action input.
- Right: party, tasks, character state.
- Public world events and personal events visually separated.

The PRD direction specifies this three-region desktop layout and mobile scene-first access to map/party/tasks via drawers.

### Mobile composition

- Scene is the first visual signal.
- Current action input remains reachable.
- Map, party, tasks, inventory, and character open through labeled drawers/tabs.
- NPC dialogue can become a full-screen mode with a clear return-to-scene control.

Do not shrink the desktop three-column panels under 1024px.

### World map

- Nodes and routes must show discovered, unknown, blocked, current, and reachable states.
- Use atmosphere and topography to create world identity, but route legality remains clear.
- Clicking a node previews destination and cost before movement confirmation.
- Provide list/route alternative for keyboard and screen readers.

### Scene narrative

- Scene title/place and time context remain visible.
- Environmental description precedes current legal action.
- World events and character dialogue use distinct structure.
- Streaming narrative has a stable reading region and does not reflow the action input on every token.

### NPC dialogue

- NPC name, portrait/subject, presence, and visibility scope at top.
- Scene/party context remains accessible.
- Public/private/queued conversation scope is explicit.
- World-changing dialogue actions require structured confirmation.
- Do not show internal model reasoning or hidden NPC knowledge.

### Free action

- Natural-language input.
- Interpretation/candidate action display.
- Legal/ambiguous/invalid outcome.
- Confirmation before world-changing result when needed.
- Authority settlement before narrative.

## 9. Cross-mode pages

### Home

- Prioritize continue play, three mode entries, official/recommended scripts, recent content.
- Mobile order: continue, modes, recommendations.
- Do not use a marketing hero that consumes the first viewport. Product content is the hero.
- An unavailable mode shows "coming later", not a dead clickable page.

### Lobby

- Search/filter region is separate from results.
- Cards expose mode, players, duration, content status, language, maturity, and action.
- Empty search explains filters and offers reset.
- Mobile filters use a drawer.

### Room creation and waiting

- Clear room type, script/world, player count, role assignment, confidentiality.
- Waiting room makes invite, readiness, host controls, and rule summary obvious.
- Chat and seat list share space without making either unreadable.

### AI workshop and editor

- Use a calmer paper/daylight visual mode.
- Three-step creation: source, target configuration, rights/content check.
- Progress distinguishes parse, semantic structure, game conversion, validation, and review.
- Editor uses object tree, structured form, issues/source, version/publish controls.
- Mobile supports viewing/light edits; complex structure editing can direct users to desktop.
- Always show private content scope and validation blocking states.

### Result

- Outcome first, causality second, comparison third.
- Use timeline, vote map, evidence use, relationship changes, and branching outcome as applicable.
- Share card includes only authorized information.
- Final reveal can be cinematic, but the persistent result page must carry all essential meaning.

### Replay

- Stage timeline, event detail, and reasoning comparison.
- On mobile, top phase switch plus collapsible event groups.
- Preserve source/provenance when displaying AI critique.
- Do not present generated narration as authoritative fact.

## 10. Theming

Use at least three practical modes:

- **Archive Night:** default for gameplay and investigation.
- **Paper Daylight:** default for script reading, creators, profiles, and editorial content.
- **Scene Override:** local environmental palette for RPG/scripted locations.

Mode themes do not replace semantic state tokens.

Example:

- RPG forest may tint the scene moss-green.
- A danger confirmation remains danger red.
- Focus remains high-contrast.
- Public/private labels remain semantically consistent.

## 11. Content and imagery

- Use actual scene/character/evidence imagery where inspection matters.
- Prefer authored or generated domain-specific scenes over generic atmospheric stock.
- Portraits need stable crop and identity.
- Evidence imagery must be legible enough to examine.
- Art-directed mobile crops must retain the subject.
- Icons remain professional and coherent; emoji are not character or clue icons.
- Missing assets have designed fallbacks and never reveal private metadata.

## 12. Mode differentiation test

A screenshot should reveal the active mode without relying only on the title:

- Murder: evidence/archive structure and investigation hierarchy.
- Werewolf: social ring/day-night/vote/phase state.
- RPG: environmental scene/map/party/narrative input.

If all three become the same sidebar plus card grid, the design has failed the mode differentiation requirement.

## 13. MysteryVerse acceptance

- Shared brand language exists.
- Each mode has distinct composition and information model.
- Operational state remains visible.
- Privacy is explicit and visually safe.
- Scene imagery carries actual place/evidence/character value.
- Motion has narrative purpose and a reduced path.
- Desktop and mobile preserve the core game loop.
- Result and replay persist meaning beyond temporary animation.
- No page falls back to generic AI-SaaS card/gradient language.
