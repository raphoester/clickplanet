import {describe, expect, it, vi} from "vitest"
import {reportClickFailure} from "./globe.ts"
import {RateLimitedError, VPNBlockedError} from "../../backends/backend.ts"
import {SessionUnavailableError} from "../../backends/session.ts"

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
