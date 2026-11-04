import {MethodIdempotency} from "@bufbuild/protobuf"
import {Code, ConnectError, StreamResponse, UnaryRequest, UnaryResponse} from "@connectrpc/connect"
import {describe, expect, it, vi} from "vitest"
import {BOOT_HEADER, Restarts, restartInterceptor} from "./restarts.ts"

function request(idempotency?: MethodIdempotency, stream = false): UnaryRequest {
    return {stream, method: {idempotency}} as unknown as UnaryRequest
}

function answeringUnder(boot: string): (req: unknown) => Promise<UnaryResponse> {
    return async () => ({stream: false, header: new Headers({[BOOT_HEADER]: boot})}) as unknown as UnaryResponse
}

function failingUnder(boot: string): (req: unknown) => Promise<UnaryResponse> {
    return async () => {
        throw new ConnectError("not found", Code.NotFound, new Headers({[BOOT_HEADER]: boot}))
    }
}

async function call(restarts: Restarts, next: (req: unknown) => Promise<UnaryResponse | StreamResponse>, req = request()) {
    return restartInterceptor(restarts)(next)(req)
}

describe("Restarts", () => {
    it("reloads once the server answers under a second boot", () => {
        const reload = vi.fn()
        const restarts = new Restarts(reload)

        restarts.heard("a")
        restarts.heard("a")
        expect(reload).not.toHaveBeenCalled()

        restarts.heard("b")
        expect(reload).toHaveBeenCalledTimes(1)
    })

    it("reloads once, whatever comes after", () => {
        const reload = vi.fn()
        const restarts = new Restarts(reload)

        restarts.heard("a")
        restarts.heard("b")
        restarts.heard("c")
        restarts.heard("a")

        expect(reload).toHaveBeenCalledTimes(1)
    })

    it("reads an answer with no boot as nothing", () => {
        const reload = vi.fn()
        const restarts = new Restarts(reload)

        restarts.heard(null)
        restarts.heard("a")
        restarts.heard(null)
        restarts.heard("")

        expect(reload).not.toHaveBeenCalled()
    })
})

describe("restartInterceptor", () => {
    it("reloads when a call is answered under a new boot", async () => {
        const reload = vi.fn()
        const restarts = new Restarts(reload)

        await call(restarts, answeringUnder("a"))
        await call(restarts, answeringUnder("b"))

        expect(reload).toHaveBeenCalledTimes(1)
    })

    it("reads the boot of a stream as it opens", async () => {
        const reload = vi.fn()
        const restarts = new Restarts(reload)

        await call(restarts, answeringUnder("a"), request(MethodIdempotency.NoSideEffects, true))
        await call(restarts, answeringUnder("b"), request(MethodIdempotency.NoSideEffects, true))

        expect(reload).toHaveBeenCalledTimes(1)
    })

    it("reads the boot of a refusal, and still throws it", async () => {
        const reload = vi.fn()
        const restarts = new Restarts(reload)

        await call(restarts, answeringUnder("a"))
        await expect(call(restarts, failingUnder("b"))).rejects.toBeInstanceOf(ConnectError)

        expect(reload).toHaveBeenCalledTimes(1)
    })

    it("does not trust a read the browser may have cached", async () => {
        const reload = vi.fn()
        const restarts = new Restarts(reload)

        await call(restarts, answeringUnder("b"))
        await call(restarts, answeringUnder("a"), request(MethodIdempotency.NoSideEffects))
        await call(restarts, failingUnder("a"), request(MethodIdempotency.NoSideEffects)).catch(() => {})

        expect(reload).not.toHaveBeenCalled()
    })
})
