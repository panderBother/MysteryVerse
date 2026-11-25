# Cinematic Scene Design

Use this reference when an interface needs to establish place, atmosphere, dramatic focus, environmental storytelling, or a transition between narrative states. The goal is not to add a background image behind text; the goal is to make the scene's layers, light, information, and interaction behave as one experience.

## 1. Scene brief

Before implementation, write:

- **Subject:** what the user is looking at and why it matters.
- **Dramatic question:** what the user should wonder or decide.
- **Time and weather:** period, hour, precipitation, visibility.
- **Place logic:** entrances, exits, architecture, terrain, objects.
- **Lighting:** source, direction, temperature, contrast.
- **Focal hierarchy:** primary object, supporting evidence, ambient context.
- **Information overlays:** what interface content must remain legible.
- **State dependency:** which conditions change the scene.
- **Transition behavior:** how the scene arrives, transforms, or leaves.

If the scene brief cannot name a focal object or dramatic question, the visual is not ready.

## 2. Layer model

Build the scene in explicit layers:

| Layer | Purpose | Typical content | Interface relationship |
| --- | --- | --- | --- |
| Ambient | Establish atmosphere | sky, fog, dust, reflected color | never interactive |
| Background | Establish place | architecture, landscape, skyline | low contrast |
| Midground | Carry dramatic subject | clue object, suspect, room focal point, path | may host hotspots |
| Foreground | Frame and create depth | railing, glass, foliage, rain, silhouette | occludes with restraint |
| Light/weather | Direct sightline | shafts, pools, rain, sparks, mist | controlled by motion budget |
| Information | Deliver task | headings, evidence, actions, status | always readable |
| Interaction | Reveal affordance | hotspots, overlays, cursor states, focus paths | rule-aware and accessible |

Do not flatten these into one image plus arbitrary overlays. Each layer needs a role, z-order, responsive crop, and fallback.

## 3. Composition

### Primary focal point

- Give the scene one unmistakable visual destination.
- Use at least two cues: scale, brightness, color temperature, sharpness, motion, placement, or line convergence.
- Keep the operational action close to the focal subject when the user is expected to act on it.
- Do not place the primary action inside a low-contrast atmospheric region.

### Sightlines

Use diagonals, corridors, doorways, tables, roads, stairs, beams, cloth, rain streaks, or relationship lines to lead the eye. Avoid centering everything or distributing equal visual weight across the screen.

### Depth

Create depth with:

- Size progression.
- Overlap and occlusion.
- Contrast falloff.
- Saturation falloff.
- Atmospheric haze.
- Parallax at very small differentials.
- Shadows that connect objects to the environment.

Parallax should be modest. Large scroll-linked parallax can harm readability and performance; use it mainly in open exploration or presentation surfaces.

### Negative space

Reserve negative space for:

- Scene narration.
- Current phase.
- Primary action.
- Critical evidence.

Do not fill negative space with floating orbs, generic lightspots, or scattered particles without a scene reason.

## 4. Lighting

Define lighting as a coherent physical story:

- **Direction:** left, right, top, back, practical source.
- **Source:** moon, window, lamp, fire, monitor, lightning, bioluminescence.
- **Temperature:** warm/cool relationship and where it contrasts.
- **Falloff:** hard, soft, volumetric, diffuse.
- **Role:** reveal focal object, hide information, create tension, signal phase.

Use gradients to model light, not as autonomous decoration. A bright source should explain the highlights on nearby objects and the shadow direction.

### Example light logic

| Scene | Key light | Fill | Accent | Desired effect |
| --- | --- | --- | --- | --- |
| Abandoned manor | cold moon through broken window | deep green-grey bounce | warm desk lamp on evidence | isolation plus investigative focus |
| Rainy street | sodium streetlamp | wet asphalt reflection | red sign or tail light | noir pressure and directional movement |
| Mysterious lab | overhead fluorescent | sterile floor bounce | cyan instrument glow | controlled unease |
| Ancient castle | torch/fire from side | cool window spill | heraldic metal glint | power and age |
| Underground chamber | directional lantern | low floor bounce | mineral/ember trace | claustrophobia and discovery |
| Crime scene | window daylight | neutral ambience | evidence amber | procedural clarity |
| Fantasy forest | canopy shafts | green ambient bounce | blue bioluminescence | wonder and unknown depth |
| Future city | reflected skyline light | glass/metal ambience | transit signal color | scale and system control |

## 5. Environment effects

### Weather

Weather must affect multiple layers:

- Rain: streaks, surface reflections, darker saturation, localized ripples, window distortion.
- Snow: reduced distant contrast, flakes at different depths, accumulation silhouettes.
- Fog: depth falloff, hidden exits, light halos, slower reveal.
- Wind: cloth, foliage, particles, sound, directional movement.

Do not place rain streaks over text without masks or backplates.

### Particles

Use particles for:

- Dust in a light beam.
- Embers near fire.
- Mist over ground.
- Spores in a forest.
- Ash after destruction.

Particle rules:

- Keep density low around reading areas.
- Use a capped count and depth-based size.
- Disable or freeze under reduced motion.
- Provide a static texture fallback.
- Do not run a full WebGL scene to render one decorative particle field.

### Texture and material

Material contrast communicates world logic:

- Wet stone, dry paper, tarnished metal, worn leather, polished glass, rough cloth.
- Dossier surfaces can carry folds, fibers, stamps, punches, and handwritten annotations.
- Metal UI suggests control and system state; paper suggests testimony and human evidence.
- Do not apply the same grain overlay to every image and panel.

## 6. Scene grammar for mystery spaces

Each mystery environment should answer:

1. What happened here?
2. What is out of place?
3. Where should the player look?
4. What is hidden by the current angle?
5. What changed since the last visit?
6. Which detail is public and which is private?

Use environmental clues:

- A chair pushed away from the table.
- Two glasses but three place settings.
- Rain entering through a window that was reported closed.
- A burned document with one readable phrase.
- Fresh mud on an interior rug.
- A portrait with a replaced face.

These details should map to actual evidence or state. Do not animate a clue-like object that the rules cannot explain.

## 7. Scene-state model

Treat the scene as data:

```ts
type CinematicSceneState = {
  sceneId: string;
  timeOfDay: "dawn" | "day" | "dusk" | "night";
  weather: "clear" | "rain" | "fog" | "storm" | "snow";
  phase: string;
  focalObjectId?: string;
  discoveredEvidenceIds: string[];
  visibleCharacterIds: string[];
  blockedExitIds: string[];
  dangerLevel: 0 | 1 | 2 | 3;
  ambientIntensity: "quiet" | "tense" | "critical";
};
```

The scene renderer should derive:

- Background variant.
- Lighting temperature.
- Foreground effects.
- Ambient sound.
- Focal hotspot treatment.
- Transition preset.

Never hardcode a visual reveal that contradicts authoritative state.

## 8. Scene transitions

### Transition types

- **Local reveal:** evidence or object gains focus without leaving the scene.
- **Phase transition:** time, weather, light, and action frame change.
- **Location transition:** move to a new place with directional continuity.
- **Perspective transition:** move from room overview to character dialogue.
- **Result transition:** collapse the scene into outcome and causality.

### Transition recipe

1. Hold or reduce competing motion.
2. Signal the cause through light, sound, text, or object motion.
3. Change the scene layer or mask.
4. Reveal the new focal region.
5. Restore controls after authoritative state is ready.
6. Preserve a persistent timestamp, map node, timeline event, or archive entry.

Do not use a generic full-screen spinner between scenes unless loading genuinely requires it.

### Interruption

Transitions must tolerate:

- Reduced motion.
- Missing high-resolution asset.
- Slow network.
- User opening a panel mid-transition.
- Reconnect during phase change.
- Duplicate state event.

## 9. Audio and haptics

If sound is available:

- Assign audio roles: ambience, causal cue, confirmation, danger, reveal.
- Keep recurring UI sounds quiet and short.
- Never use sound as the only signal.
- Provide master, ambience, and effects controls.
- Provide text captions for meaningful non-speech cues.
- Respect autoplay restrictions; start only after user interaction when required.
- On mobile, use haptics only for confirmed actions and allow them to be disabled.

Sound and visual timing should agree, but visual information must remain complete with sound off.

## 10. Asset strategy

Choose the smallest asset that supports inspection:

- **Bitmap scene:** when place, texture, object, or person must be inspected.
- **Layered CSS:** when masks, panels, paper, lighting, and simple depth suffice.
- **Canvas:** for dense weather, particles, maps, or dynamic relationship lines.
- **Three.js/WebGL:** only when spatial navigation, true 3D objects, camera movement, or physical depth materially improves the task.

Rules:

- Use real or generated domain-relevant imagery rather than vague atmospheric stock.
- Export responsive widths and modern formats.
- Provide art-directed crops for mobile instead of blind center-cropping.
- Provide dominant-color placeholders and explicit loading failure states.
- Do not use blurry imagery when users must inspect evidence.
- Keep scene text separate from raster assets for localization and accessibility.

## 11. Performance tiers

Define three tiers:

### Tier A: full

- Layered scene image or modest 3D.
- Ambient particles.
- Foreground weather.
- Sound.
- Cinematic transitions.

### Tier B: balanced

- Static layered scene.
- CSS light/weather effects.
- Reduced particle count.
- Shorter transitions.

### Tier C: reduced/static

- Static scene image or designed gradient/paper composition.
- Opacity-only transitions.
- No parallax, particles, shaking, or full-screen motion.
- Persistent text and state cues.

Detect capability only when needed and always allow the user to request reduced motion independently of hardware.

## 12. Scene review checklist

- Is there a clear focal subject?
- Does the light direction explain highlights and shadows?
- Are foreground, midground, and background distinct?
- Does the scene remain readable under UI overlays?
- Does every effect support place, state, or emotion?
- Are mobile crops intentional?
- Are private details absent from public scene layers?
- Does the scene change when game state changes?
- Is there a static fallback?
- Are assets inspectable rather than obscured?
- Does the scene leave a persistent artifact after a major reveal?
- Is the scene still usable with sound and ambient motion off?
