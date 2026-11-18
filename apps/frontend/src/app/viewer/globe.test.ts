import {describe, expect, it, vi} from "vitest"
import {drawsFrame, reportClaimFailure, reportClickFailure, spins, spinStep} from "./globe.ts"
import {MIN_ZOOM, RESTING_ZOOM} from "./zoom.ts"
import {AllegianceLockedError, BannedError, BonusLostError, FrozenError, OutOfReachError, RateLimitedError, VPNBlockedError} from "../../backends/backend.ts"
import {SessionUnavailableError} from "../../backends/session.ts"
import {OwnerChange, TileOwnership} from "../../domain/tileOwnership.ts"
import {TileField} from "./tileField.ts"
import {chunksOf} from "../../domain/tileChunks.ts"
import {threeLattice} from "../../domain/lattice.ts"
import {regions} from "./atlas.ts"
import {PLAIN} from "./graphics.ts"

function handlers() {
    return {onRateLimited: vi.fn(), onFrozen: vi.fn(), onVPNBlocked: vi.fn(), onBanned: vi.fn(), onSessionUnavailable: vi.fn(), onAllegianceLocked: vi.fn(), onOutOfReach: vi.fn()}
}

describe("reportClickFailure", () => {
    it("sends the throttle's refusal to the meter, and to no dialog", () => {
        const h = handlers()

        expect(reportClickFailure(new RateLimitedError(), h)).toBe(true)

        expect(h.onRateLimited).toHaveBeenCalledTimes(1)
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
    })

    it("sends a click on a frozen map to the meter, and to no dialog", () => {
        const h = handlers()

        expect(reportClickFailure(new FrozenError(), h)).toBe(true)

        expect(h.onFrozen).toHaveBeenCalledTimes(1)
        expect(h.onRateLimited).not.toHaveBeenCalled()
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
    })

    it("sends a click out of the flag's reach to the reach bubble, and to nothing else", () => {
        const h = handlers()

        expect(reportClickFailure(new OutOfReachError(), h)).toBe(true)

        expect(h.onOutOfReach).toHaveBeenCalledTimes(1)
        expect(h.onFrozen).not.toHaveBeenCalled()
        expect(h.onRateLimited).not.toHaveBeenCalled()
    })

    it("sends the VPN refusal to the VPN dialog, and nowhere else", () => {
        const h = handlers()

        expect(reportClickFailure(new VPNBlockedError(), h)).toBe(true)

        expect(h.onVPNBlocked).toHaveBeenCalledTimes(1)
        expect(h.onRateLimited).not.toHaveBeenCalled()
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
    })

    it("sends a ban to its own dialog with its end, and not to the VPN one", () => {
        const h = handlers()
        const until = Date.now() + 3600_000

        expect(reportClickFailure(new BannedError(until), h)).toBe(true)

        expect(h.onBanned).toHaveBeenCalledWith(until)
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
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

    it("says which flag a click was refused for and until when, with no dialog and no log", () => {
        const h = handlers()
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})

        expect(reportClickFailure(new AllegianceLockedError({countryId: "de", changeableAt: 1}), h)).toBe(true)

        expect(h.onAllegianceLocked).toHaveBeenCalledWith({countryId: "de", changeableAt: 1})
        expect(logged).not.toHaveBeenCalled()
        expect(h.onRateLimited).not.toHaveBeenCalled()
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
        logged.mockRestore()
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

describe("reportClaimFailure", () => {
    it("sends a bomb or a shield refused for a ban to the ban dialog", () => {
        const h = {onSessionUnavailable: vi.fn(), onBanned: vi.fn()}
        const until = Date.now() + 3600_000

        reportClaimFailure(new BannedError(until), h)

        expect(h.onBanned).toHaveBeenCalledWith(until)
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
    })

    it("says nothing of a bonus that was lost", () => {
        const h = {onSessionUnavailable: vi.fn(), onBanned: vi.fn()}

        reportClaimFailure(new BonusLostError(), h)

        expect(h.onBanned).not.toHaveBeenCalled()
        expect(h.onSessionUnavailable).not.toHaveBeenCalled()
    })
})

describe("a refused click, from the paint to the rollback", () => {
    const size = 32

    function wiring() {
        const positions = new Float32Array(size * 3).map((_, i) => (i % 3 === 0 ? 1 : 0))
        const uniforms = {pointSize: {value: 1}, pixelsPerRadian: {value: 1}, flagPaint: {value: 0}, blastRadii: {value: [0]}, motion: {value: 1}}
        const field = new TileField(uniforms, {
            positions,
            size,
            cells: {lattice: threeLattice(1), words: new Uint32Array(2), texels: Uint32Array.from({length: size}, (_, i) => i), tiles: Uint32Array.from({length: size}, (_, i) => i)},
            chunks: chunksOf(positions),
        }, PLAIN)
        const ownership = new TileOwnership(size)

        const applyChanges = (changes: OwnerChange[]) => {
            if (changes.length > 0) field.setOwners(changes)
        }

        const live = (tile: number, country: string) => {
            const changes: OwnerChange[] = []
            ownership.set(tile, country, changes)
            applyChanges(changes)
        }

        const regionOf = (tile: number) => {
            const [at, size] = (field.displayPoints.geometry.getAttribute("flagRegion").array as Float32Array).slice((tile - 1) * 2, tile * 2)
            return [at % 2048, Math.floor(at / 2048), size % 2048, Math.floor(size / 2048)]
        }

        return {ownership, applyChanges, live, regionOf}
    }

    it("paints the tile, then blanks it again", () => {
        const {ownership, applyChanges, regionOf} = wiring()
        const fr = regions.get("fr")!

        const {changes, claim} = ownership.applyOptimistic(3, "fr")
        applyChanges(changes)

        expect(regionOf(3)).toEqual([fr.x, fr.y, fr.width, fr.height])

        applyChanges(ownership.rollback(claim))

        expect(regionOf(3)).toEqual([0, 0, 0, 0])
    })

    it("gives the tile back to the country that held it", () => {
        const {ownership, applyChanges, live, regionOf} = wiring()
        const jp = regions.get("jp")!

        live(3, "jp")

        const {changes, claim} = ownership.applyOptimistic(3, "fr")
        applyChanges(changes)
        applyChanges(ownership.rollback(claim))

        expect(regionOf(3)).toEqual([jp.x, jp.y, jp.width, jp.height])
    })

    it("leaves a click the server had already echoed alone", () => {
        const {ownership, applyChanges, live, regionOf} = wiring()
        const fr = regions.get("fr")!

        const {changes, claim} = ownership.applyOptimistic(3, "fr")
        applyChanges(changes)
        live(3, "fr")

        applyChanges(ownership.rollback(claim))

        expect(regionOf(3)).toEqual([fr.x, fr.y, fr.width, fr.height])
    })
})

describe("drawsFrame", () => {
    const tick = (over: Partial<Parameters<typeof drawsFrame>[0]>) => drawsFrame({
        turned: false, changed: false, ambient: false, at: 10_000, drawnAt: 0, interactingUntil: 0,
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

    it("takes the display's nearest frame rather than the first one past the cap", () => {
        const aWhiskerShort = {turned: true, at: 10_000, drawnAt: 9_984.1, sinceLastTick: 16.7}

        expect(tick(aWhiskerShort)).toBe(true)
    })

    it("still holds back the frame that lands halfway to the cap", () => {
        const halfway = {turned: true, at: 10_000, drawnAt: 9_991.67, sinceLastTick: 8.33}

        expect(tick(halfway)).toBe(false)
    })

    it("draws a pulse at a slower pace than the spin", () => {
        expect(tick({ambient: true, drawnAt: 9_960})).toBe(false)
        expect(tick({ambient: true, drawnAt: 9_950})).toBe(true)
        expect(tick({ambient: true, turned: true, drawnAt: 9_984})).toBe(true)
    })

    it("never holds a change back for the spin's sake", () => {
        expect(tick({turned: true, changed: true, drawnAt: 9_999})).toBe(true)
    })
})

describe("spins", () => {
    const spin = (over: Partial<Parameters<typeof spins>[0]>) => spins({
        zoom: RESTING_ZOOM, lastInputAt: 0, now: 0, ...over,
    })

    it("turns the resting globe from the page's load", () => {
        expect(spin({})).toBe(true)
    })

    it("keeps turning for the minute after the last input", () => {
        expect(spin({lastInputAt: 1_000, now: 60_999})).toBe(true)
    })

    it("stops once nobody has touched the page for a minute", () => {
        expect(spin({lastInputAt: 1_000, now: 61_000})).toBe(false)
    })

    it("turns again from the next input", () => {
        expect(spin({lastInputAt: 120_000, now: 120_016})).toBe(true)
    })

    it("never turns a globe zoomed in past the resting zoom", () => {
        expect(spin({zoom: RESTING_ZOOM + 0.01})).toBe(false)
    })

    it("turns a globe zoomed out past the resting zoom", () => {
        expect(spin({zoom: MIN_ZOOM})).toBe(true)
    })
})

describe("spinStep", () => {
    it("advances the spin by however long the frame took", () => {
        expect(spinStep(16.7)).toBeCloseTo(0.0167, 5)
    })

    it("does not carry a hidden tab's whole absence into one frame", () => {
        expect(spinStep(15 * 60 * 1000)).toBe(0.1)
    })

    it("never turns the globe backwards", () => {
        expect(spinStep(-5)).toBe(0)
    })
})
