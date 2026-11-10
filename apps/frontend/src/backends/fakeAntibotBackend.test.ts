import {describe, expect, it, vi} from "vitest"
import {FakeAntibotBackend} from "./fakeAntibotBackend.ts"

describe("FakeAntibotBackend", () => {
    it("tells a warning live, and again to each stream that opens until it is acknowledged", async () => {
        const fake = new FakeAntibotBackend()
        const live = vi.fn()
        fake.listenForWarnings(live)

        fake.warn()
        const opened = vi.fn()
        fake.listenForWarnings(opened)
        await fake.acknowledgeWarning()
        const later = vi.fn()
        fake.listenForWarnings(later)

        expect(live).toHaveBeenCalledTimes(1)
        expect(opened).toHaveBeenCalledTimes(1)
        expect(later).not.toHaveBeenCalled()
    })

    it("tells nothing to a stream it stopped", () => {
        const fake = new FakeAntibotBackend()
        const onWarned = vi.fn()
        const stop = fake.listenForWarnings(onWarned)

        stop()
        fake.warn()

        expect(onWarned).not.toHaveBeenCalled()
    })
})
