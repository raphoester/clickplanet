import {describe, expect, it, vi} from "vitest"
import {reportClickFailure} from "./globe.ts"
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

// Three refusals give three different pieces of advice — ease off, turn the VPN
// off, reload or unblock the challenge — so sending one to the wrong dialog
// leaves a working page telling the player to fix the wrong thing.
describe("reportClickFailure", () => {
    it("sends the throttle's refusal to the throttle's dialog, and nowhere else", () => {
        const h = handlers()

        reportClickFailure(new RateLimitedError(), h)

        expect(h.onRateLimited).toHaveBeenCalledTimes(1)
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
    })

    it("sends the VPN refusal to the VPN dialog, and nowhere else", () => {
        const h = handlers()

        reportClickFailure(new VPNBlockedError(), h)

        expect(h.onVPNBlocked).toHaveBeenCalledTimes(1)
        expect(h.onRateLimited).not.toHaveBeenCalled()
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
    })

    it("sends a session that could not be obtained to its own dialog, and nowhere else", () => {
        const h = handlers()

        reportClickFailure(new SessionUnavailableError(), h)

        expect(h.onSessionUnavailable).toHaveBeenCalledTimes(1)
        expect(h.onRateLimited).not.toHaveBeenCalled()
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
    })

    it("logs anything else and raises no dialog", () => {
        const h = handlers()
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})

        reportClickFailure(new Error("the network fell over"), h)

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
        const field = new TileField({zoom: {value: 1}}, {
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
