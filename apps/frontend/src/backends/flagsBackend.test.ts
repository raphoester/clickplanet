import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError} from "@connectrpc/connect"
import {Timestamp} from "@bufbuild/protobuf"
import {ConnectFlagsBackend, flagChangedOf, listedOf} from "./flagsBackend.ts"
import {Flag, FlagEvent, Heartbeat, ListFlagsResponse, Picture, Pin, Sold} from "../gen/grpc/flags/v1/flags_pb.ts"

const heartbeat = () => new FlagEvent({event: {case: "heartbeat", value: new Heartbeat()}})
const changed = (code: string, name: string, closed = false) =>
    new FlagEvent({event: {case: "flagChanged", value: new Flag({code, name, closed})}})

const backendOf = (listenForEvents: ReturnType<typeof vi.fn>, listFlags = vi.fn(), fresh = vi.fn()) =>
    new ConnectFlagsBackend({listFlags, listenForEvents} as never, {listFlags: fresh, listenForEvents: vi.fn()} as never)

describe("flagChangedOf", () => {
    it("reads a flag opened or closed, and nothing from a heartbeat or a case this build does not know", () => {
        expect(flagChangedOf(changed("r-k3p9x2ma0b", "Bavaria", true)))
            .toEqual({code: "r-k3p9x2ma0b", name: "Bavaria", closed: true})
        expect(flagChangedOf(heartbeat())).toBeUndefined()
        expect(flagChangedOf(new FlagEvent())).toBeUndefined()
    })
})

describe("listedOf", () => {
    it("reads a sold flag's picture and its times, in milliseconds", () => {
        const flag = new Flag({
            code: "r-k3p9x2ma0b", name: "Bavaria", closed: true,
            sold: new Sold({
                picture: new Picture({url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: true, focusX: 0.25, focusY: 0.5}),
                endsAt: Timestamp.fromDate(new Date(2_000)),
                closedAt: Timestamp.fromDate(new Date(3_000)),
            }),
        })
        expect(listedOf(flag)).toEqual({
            code: "r-k3p9x2ma0b", name: "Bavaria", closed: true,
            sold: {
                picture: {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: true, focusX: 0.25, focusY: 0.5},
                endsAt: 2_000, closedAt: 3_000,
            },
        })
    })

    it("reads a sold flag's pin, where it starts in Capital Siege", () => {
        const flag = new Flag({
            code: "r-k3p9x2ma0b", name: "Bavaria",
            sold: new Sold({picture: new Picture({url: "x.png", width: 2, height: 1}), pin: new Pin({latitude: 48.1, longitude: 11.6})}),
        })
        expect(listedOf(flag).sold?.pin).toEqual({latitude: 48.1, longitude: 11.6})
        expect(listedOf(new Flag({code: "r-k3p9x2ma0b", name: "Bavaria", sold: new Sold({picture: new Picture({url: "x.png"})})})).sold?.pin)
            .toBeUndefined()
    })

    it("reads a built flag, and an open sold one with no close", () => {
        expect(listedOf(new Flag({code: "fr", name: "France"}))).toEqual({code: "fr", name: "France", closed: false})
        const open = listedOf(new Flag({code: "r-k3p9x2ma0b", name: "Bavaria", sold: new Sold({picture: new Picture({url: "x.png", width: 2, height: 1}), endsAt: Timestamp.fromDate(new Date(5))})}))
        expect(open.sold?.closedAt).toBeUndefined()
        expect(open.sold?.endsAt).toBe(5)
    })
})

describe("ConnectFlagsBackend", () => {
    it("says the stream opened on each connection's first message, the first one included, and tells each flag", async () => {
        const listenForEvents = vi.fn()
            .mockImplementationOnce(async function* () {
                yield heartbeat()
                yield changed("r-k3p9x2ma0b", "Bavaria")
                yield new FlagEvent()
            })
            .mockImplementationOnce(async function* () {
                yield heartbeat()
                await new Promise(() => {})
            })
        const onOpened = vi.fn()
        const onChanged = vi.fn()
        const stop = backendOf(listenForEvents).listenForFlags({onOpened, onChanged})

        await vi.waitFor(() => expect(onChanged).toHaveBeenCalledOnce())
        expect(onChanged).toHaveBeenCalledWith({code: "r-k3p9x2ma0b", name: "Bavaria", closed: false})
        expect(onOpened).toHaveBeenCalledOnce()

        await vi.waitFor(() => expect(onOpened).toHaveBeenCalledTimes(2), {timeout: 3_000})
        expect(listenForEvents).toHaveBeenCalledTimes(2)
        expect(onChanged).toHaveBeenCalledOnce()
        stop()
    })

    it("stops for good on a server with no flags module, rather than reconnecting", async () => {
        const listenForEvents = vi.fn(async function* () {
            yield* []
            throw new ConnectError("no flags", Code.Unimplemented)
        })
        const onOpened = vi.fn()
        backendOf(listenForEvents).listenForFlags({onOpened, onChanged: vi.fn()})

        await vi.waitFor(() => expect(listenForEvents).toHaveBeenCalledOnce())
        await new Promise((resolve) => setTimeout(resolve, 800))

        expect(listenForEvents).toHaveBeenCalledOnce()
        expect(onOpened).not.toHaveBeenCalled()
    })

    it("reads the list past the cache when asked fresh, and lists none on a server with no flags module", async () => {
        const listFlags = vi.fn().mockRejectedValue(new ConnectError("no flags", Code.Unimplemented))
        const fresh = vi.fn().mockResolvedValue(new ListFlagsResponse({flags: [new Flag({code: "fr", name: "France"})]}))
        const backend = backendOf(vi.fn(), listFlags, fresh)

        expect(await backend.listFlags(false)).toEqual([])
        expect(await backend.listFlags(true)).toEqual([{code: "fr", name: "France", closed: false}])
    })
})
