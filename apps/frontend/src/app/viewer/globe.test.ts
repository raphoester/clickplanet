import {describe, expect, it, vi} from "vitest"
import {reportClickFailure} from "./globe.ts"
import {RateLimitedError, VPNBlockedError} from "../../backends/backend.ts"

function handlers() {
    return {onRateLimited: vi.fn(), onVPNBlocked: vi.fn()}
}

describe("reportClickFailure", () => {
    it("sends the throttle's refusal to the throttle's dialog, and nowhere else", () => {
        const h = handlers()

        reportClickFailure(new RateLimitedError(), h)

        expect(h.onRateLimited).toHaveBeenCalledTimes(1)
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
    })

    it("sends the VPN refusal to the VPN dialog, and nowhere else", () => {
        const h = handlers()

        reportClickFailure(new VPNBlockedError(), h)

        expect(h.onVPNBlocked).toHaveBeenCalledTimes(1)
        expect(h.onRateLimited).not.toHaveBeenCalled()
    })

    /**
     * A real fault still belongs in the console. Showing the player a dialog
     * for it would be worse than saying nothing: they cannot act on it.
     */
    it("logs anything else and raises no dialog", () => {
        const h = handlers()
        const logged = vi.spyOn(console, "error").mockImplementation(() => {})

        reportClickFailure(new Error("the network fell over"), h)

        expect(logged).toHaveBeenCalledTimes(1)
        expect(h.onRateLimited).not.toHaveBeenCalled()
        expect(h.onVPNBlocked).not.toHaveBeenCalled()
        logged.mockRestore()
    })
})
