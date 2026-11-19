import {describe, expect, it, vi} from "vitest"
import {beatClock, drawsFrame, reportClaimFailure, reportClickFailure, spins, spinStep, watched} from "./globe.ts"
import {MIN_ZOOM, RESTING_ZOOM} from "./zoom.ts"
import {AllegianceLockedError, BannedError, BonusLostError, FrozenError, OutOfReachError, RateLimitedError, VPNBlockedError} from "../../backends/backend.ts"
import {SessionUnavailableError} from "../../backends/session.ts"

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

describe("drawsFrame", () => {
    const tick = (over: Partial<Parameters<typeof drawsFrame>[0]>) => drawsFrame({
        turned: false, changed: false, glints: "none", ambient: false, attended: true, at: 10_000, drawnAt: 0,
        interactingUntil: 0, sinceLastTick: 0, ...over,
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

    it("draws a glint at the spin's pace while the page is in use", () => {
        expect(tick({glints: "playing", drawnAt: 9_990})).toBe(false)
        expect(tick({glints: "playing", drawnAt: 9_984})).toBe(true)
    })

    it("draws every other frame of a glint on a 120Hz display", () => {
        const ticks = [8.33, 16.67, 25, 33.33, 41.67, 50].map((at) => 10_000 + at)
        let drawnAt = 10_000
        const drawn = ticks.filter((at) => {
            const draws = tick({glints: "playing", at, drawnAt, sinceLastTick: 8.33})
            if (draws) drawnAt = at
            return draws
        })

        expect(drawn).toEqual([10_016.67, 10_033.33, 10_050])
    })

    it("draws every frame of a glint while the player handles the globe", () => {
        expect(tick({glints: "playing", drawnAt: 9_999, interactingUntil: 10_000})).toBe(true)
    })

    it("draws a glint at the pulse's pace once nobody touches the page", () => {
        expect(tick({glints: "playing", attended: false, drawnAt: 9_960})).toBe(false)
        expect(tick({glints: "playing", attended: false, drawnAt: 9_950})).toBe(true)
    })

    it("takes the last glint off the screen at once, even on a page nobody touches", () => {
        expect(tick({glints: "over", attended: false, drawnAt: 9_999})).toBe(true)
    })

    it("never holds a change back for a glint's sake", () => {
        expect(tick({glints: "playing", attended: false, changed: true, drawnAt: 9_999})).toBe(true)
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

describe("watched", () => {
    it("holds for the minute after the last input", () => {
        expect(watched({lastInputAt: 1_000, now: 60_999})).toBe(true)
    })

    it("lapses once nobody has touched the page for a minute", () => {
        expect(watched({lastInputAt: 1_000, now: 61_000})).toBe(false)
    })
})

describe("beatClock", () => {
    const shadersBeat = (clock: number) => 0.5 + 0.5 * Math.sin(clock * 2 * Math.PI / 1.1)
    const unattended = (clock: number, ticks: number) => {
        for (let i = 0; i < ticks; i++) clock = beatClock({clock, step: 0.05, attended: false})
        return clock
    }

    it("beats with the frame while the page is watched", () => {
        expect(beatClock({clock: 3, step: 0.05, attended: true})).toBeCloseTo(3.05, 9)
    })

    it("past the quiet minute, beats on to its faintest within one beat, wherever it was", () => {
        for (const start of [0, 0.4, 0.826, 2.5]) {
            const held = unattended(start, 100)

            expect(shadersBeat(held)).toBeCloseTo(0, 9)
            expect(held - start).toBeLessThanOrEqual(1.1)
        }
    })

    it("holds there for good, however long nobody watches", () => {
        const held = unattended(0, 100)

        expect(unattended(held, 100_000)).toBe(held)
    })

    it("beats on from where it held at the next input", () => {
        const held = unattended(0, 100)

        expect(beatClock({clock: held, step: 0.05, attended: true})).toBeCloseTo(held + 0.05, 9)
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
