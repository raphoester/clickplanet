import {describe, expect, it, vi} from "vitest"
import {drawsFrame, reportClickFailure, spinStep} from "./globe.ts"
import {RateLimitedError, VPNBlockedError} from "../../backends/backend.ts"
import {SessionUnavailableError} from "../../backends/session.ts"
import {OwnerChange, TileOwnership} from "../../domain/tileOwnership.ts"
import {LeaderboardEntry, rankCountries} from "../../domain/leaderboard.ts"
import {Countries} from "../../domain/countries.ts"
import {TileField} from "./tileField.ts"
import {regions} from "./atlas.ts"

function handlers() {
    return {onRateLimited: vi.fn(), onVPNBlocked: vi.fn(), onSessionUnavailable: vi.fn()}
}

// Three refusals mean three different things — ease off (said by the meter),
// turn the VPN off, reload or unblock the challenge — so sending one to the
// wrong place leaves a working page telling the player to fix the wrong thing.
describe("reportClickFailure", () => {
    it("sends the throttle's refusal to the meter, and to no dialog", () => {
        const h = handlers()

        expect(reportClickFailure(new RateLimitedError(), h)).toBe(true)

        expect(h.onRateLimited).toHaveBeenCalledTimes(1)
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
    })

    it("sends the VPN refusal to the VPN dialog, and nowhere else", () => {
        const h = handlers()

        expect(reportClickFailure(new VPNBlockedError(), h)).toBe(true)

        expect(h.onVPNBlocked).toHaveBeenCalledTimes(1)
        expect(h.onRateLimited).not.toHaveBeenCalled()
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
    })

    it("sends a session that could not be obtained to its own dialog, and nowhere else", () => {
        const h = handlers()

        expect(reportClickFailure(new SessionUnavailableError(), h)).toBe(true)

        expect(h.onSessionUnavailable).toHaveBeenCalledTimes(1)
        expect(h.onRateLimited).not.toHaveBeenCalled()
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
    })

    it("logs anything else and raises no dialog", () => {
        const h = handlers()
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})

        expect(reportClickFailure(new Error("the network fell over"), h)).toBe(false)

        expect(logged).toHaveBeenCalledTimes(1)
        expect(h.onRateLimited).not.toHaveBeenCalled()
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
        logged.mockRestore()
    })
})

// What the click handler does with the three pieces it wires together, minus the
// GPU pick: paint the click, then take it back when the server refuses it. The
// pick itself needs a real WebGL context, so this reproduces the wiring rather
// than driving `createGlobe`.
describe("a refused click, from the paint to the rollback", () => {
    const size = 32

    function wiring() {
        const field = new TileField({zoom: {value: 1}}, {pointSize: {value: 1}}, {
            positions: new Float32Array(size * 3),
            uvs: new Float32Array(size * 2),
            size,
        })
        const ownership = new TileOwnership(size)
        let leaderboard: LeaderboardEntry[] = []

        const applyChanges = (changes: OwnerChange[]) => {
            if (changes.length === 0) return
            field.setOwners(changes)
            leaderboard = rankCountries(ownership.counts())
        }

        const regionOf = (tile: number) => Array.from(
            (field.displayPoints.geometry.getAttribute("regionVector").array as Float32Array)
                .slice((tile - 1) * 4, tile * 4))

        return {ownership, applyChanges, regionOf, tiles: () => leaderboard}
    }

    it("paints the tile and ranks it, then blanks it and drops it again", () => {
        const {ownership, applyChanges, regionOf, tiles} = wiring()
        const fr = regions.get("fr")!

        const {changes, claim} = ownership.applyOptimistic(3, "fr")
        applyChanges(changes)

        expect(regionOf(3)).toEqual([fr.x, fr.y, fr.width, fr.height])
        expect(tiles()).toEqual([{country: Countries.get("fr"), tiles: 1}])

        applyChanges(ownership.rollback(claim))

        expect(regionOf(3)).toEqual([0, 0, 0, 0])
        expect(tiles()).toEqual([])
    })

    it("gives the tile and the rank back to the country that held it", () => {
        const {ownership, applyChanges, regionOf, tiles} = wiring()
        const jp = regions.get("jp")!

        applyChanges(ownership.applyUpdates([{tile: 3, previousCountry: undefined, newCountry: "jp"}]))

        const {changes, claim} = ownership.applyOptimistic(3, "fr")
        applyChanges(changes)
        applyChanges(ownership.rollback(claim))

        expect(regionOf(3)).toEqual([jp.x, jp.y, jp.width, jp.height])
        expect(tiles()).toEqual([{country: Countries.get("jp"), tiles: 1}])
    })

    it("leaves a click the server had already echoed alone", () => {
        const {ownership, applyChanges, regionOf, tiles} = wiring()
        const fr = regions.get("fr")!

        const {changes, claim} = ownership.applyOptimistic(3, "fr")
        applyChanges(changes)
        applyChanges(ownership.applyUpdates([{tile: 3, previousCountry: undefined, newCountry: "fr"}]))

        applyChanges(ownership.rollback(claim))

        expect(regionOf(3)).toEqual([fr.x, fr.y, fr.width, fr.height])
        expect(tiles()).toEqual([{country: Countries.get("fr"), tiles: 1}])
    })
})

// The globe used to redraw the same picture sixty times a second for as long
// as the tab was open. What decides that now is one function, so the rule can
// be read and pinned without a GPU.
describe("drawsFrame", () => {
    const tick = (over: Partial<Parameters<typeof drawsFrame>[0]>) => drawsFrame({
        turned: false, changed: false, at: 10_000, drawnAt: 0, interactingUntil: 0,
        sinceLastTick: 0, ...over,
    })

    it("draws nothing while the globe sits still", () => {
        expect(tick({})).toBe(false)
    })

    it("draws whatever changed, however long ago the last frame was", () => {
        expect(tick({changed: true, drawnAt: 9_999})).toBe(true)
    })

    it("draws every frame of the idle spin the player is still handling", () => {
        expect(tick({turned: true, drawnAt: 9_999, interactingUntil: 10_000})).toBe(true)
    })

    it("paces the idle spin once the globe is let go", () => {
        expect(tick({turned: true, drawnAt: 9_990})).toBe(false)
        expect(tick({turned: true, drawnAt: 9_960})).toBe(true)
    })

    // A 16ms cap against a 16.7ms frame is decided by a fraction of a
    // millisecond. This frame is a tenth of one short of the cap, and asking
    // for the first frame strictly past it would give up the whole of the next
    // one: the spin came out 60fps, then 40, then 60 again. Unevenness is seen
    // where the rate itself is not, so the nearest frame wins by a whisker.
    it("takes the display's nearest frame rather than the first one past the cap", () => {
        const aWhiskerShort = {turned: true, at: 10_000, drawnAt: 9_984.1, sinceLastTick: 16.7}

        expect(tick(aWhiskerShort)).toBe(true)
    })

    // The other side of the same rule: half a frame of slack, and no more. On a
    // 120Hz display the cap is worth two of its frames, and the one in between
    // is still held back.
    it("still holds back the frame that lands halfway to the cap", () => {
        const halfway = {turned: true, at: 10_000, drawnAt: 9_991.67, sinceLastTick: 8.33}

        expect(tick(halfway)).toBe(false)
    })

    // A tile claimed, or a blast, is never held back a frame to pace the spin.
    it("never holds a change back for the spin's sake", () => {
        expect(tick({turned: true, changed: true, drawnAt: 9_999})).toBe(true)
    })
})

// The spin is advanced by the clock rather than by a count of frames, so that
// it is the same speed on every display. That leaves it exposed to a gap in the
// frames the browser offers, which is what this is about.
describe("spinStep", () => {
    it("advances the spin by however long the frame took", () => {
        expect(spinStep(16.7)).toBeCloseTo(0.0167, 5)
    })

    // A hidden tab is offered no frames at all, and the whole of the wait
    // arrives as the first tick back. Carried through, the globe would be
    // somewhere else by the time it is looked at again.
    it("does not carry a hidden tab's whole absence into one frame", () => {
        expect(spinStep(15 * 60 * 1000)).toBe(0.1)
    })

    // `performance.now()` does not go backwards, but a first tick has nothing
    // to subtract from, and a negative step would turn the globe the wrong way.
    it("never turns the globe backwards", () => {
        expect(spinStep(-5)).toBe(0)
    })
})
