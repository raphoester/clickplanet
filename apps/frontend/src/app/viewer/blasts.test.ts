import {describe, expect, it} from "vitest"
import {BLAST_TIMELINE} from "../../domain/blast.ts"
import {flashAt, incomingAt} from "./blasts.ts"

const RADIUS = 0.06

describe("incomingAt", () => {
    it("closes in on the target and brightens while the bomb falls", () => {
        const early = incomingAt(0.05)
        const late = incomingAt(BLAST_TIMELINE.fall - 0.05)

        expect(late.reach).toBeLessThan(early.reach)
        expect(late.opacity).toBeGreaterThan(early.opacity)
    })

    it("is gone once the bomb has landed", () => {
        expect(incomingAt(BLAST_TIMELINE.fall).reach).toBe(0)
    })
})

describe("flashAt", () => {
    it("draws nothing before the drop", () => {
        expect(flashAt(-0.1, RADIUS).opacity).toBe(0)
    })

    it("swells while the bomb falls", () => {
        const early = flashAt(0.1, RADIUS)
        const late = flashAt(BLAST_TIMELINE.fall - 0.05, RADIUS)

        expect(late.scale).toBeGreaterThan(early.scale)
        expect(late.opacity).toBeGreaterThan(early.opacity)
    })

    it("bursts far wider than the crater on impact, then fades out", () => {
        const impact = flashAt(BLAST_TIMELINE.fall + 0.1, RADIUS)
        expect(impact.scale).toBeGreaterThan(RADIUS * 3)

        expect(flashAt(BLAST_TIMELINE.fall + 0.6, RADIUS).opacity).toBeLessThan(impact.opacity)
        expect(flashAt(BLAST_TIMELINE.fall + 2, RADIUS).opacity).toBe(0)
    })
})
