import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

import type {
  BusinessErrorCode,
  GameEventType,
  PhaseChanged,
  WebSocketEnvelope,
} from "../src/generated/contracts";

interface ManifestEntry {
  definition: string;
  file: string;
}

interface ContractManifest {
  examples: ManifestEntry[];
}

const root = `${resolve(process.cwd(), "../..")}/`;
const schema = JSON.parse(
  readFileSync(`${root}packages/contracts/json-schema/mysteryverse.schema.json`, "utf8"),
) as {
  $defs: Record<string, unknown>;
  $schema: string;
};
const manifest = JSON.parse(
  readFileSync(`${root}packages/contracts/examples/manifest.json`, "utf8"),
) as ContractManifest;

describe("generated contracts", () => {
  it.each(manifest.examples)("validates $definition example $file", ({ definition, file }) => {
    const ajv = new Ajv2020({ allErrors: true, strict: false });
    addFormats(ajv);

    const validator = ajv.compile({
      $schema: schema.$schema,
      $ref: `#/$defs/${definition}`,
      $defs: schema.$defs,
    });
    const document = JSON.parse(
      readFileSync(`${root}packages/contracts/examples/${file}`, "utf8"),
    ) as unknown;

    expect(validator(document)).toBe(true);

    if (definition === "WebSocketEnvelope") {
      const envelope = document as WebSocketEnvelope;
      if (envelope.stream === "game" || envelope.stream === "ai_stream") {
        expect(envelope.game_id).toBeTypeOf("string");
      } else {
        expect(envelope).not.toHaveProperty("game_id");
      }
    }
  });

  it("keeps enum values aligned", () => {
    const eventTypes: GameEventType[] = [
      "game.created",
      "room.player_joined",
      "room.player_left",
      "game.started",
      "game.phase_changed",
      "game.clue_unlocked",
      "game.message_received",
      "game.vote_submitted",
      "game.paused",
      "game.resumed",
      "game.ended",
    ];
    const errorCodes: BusinessErrorCode[] = [
      "VALIDATION_ERROR",
      "STATE_VERSION_CONFLICT",
      "AI_SAFETY_BLOCKED",
    ];
    const schemaEventTypes = (schema.$defs["GameEventType"] as { enum: GameEventType[] }).enum;

    expect(eventTypes).toHaveLength(11);
    expect(new Set(eventTypes)).toEqual(new Set(schemaEventTypes));
    expect(errorCodes).toContain("STATE_VERSION_CONFLICT");
  });

  it("round-trips the phase event with a nullable deadline", () => {
    const phase = JSON.parse(
      readFileSync(`${root}packages/contracts/examples/phase-changed.json`, "utf8"),
    ) as PhaseChanged;
    const roundTrip = JSON.parse(JSON.stringify(phase)) as PhaseChanged;

    expect(roundTrip.deadline_at).toBeNull();
    expect(roundTrip.available_actions).toEqual(["search", "chat"]);
  });
});
