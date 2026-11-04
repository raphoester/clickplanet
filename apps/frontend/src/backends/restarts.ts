import {MethodIdempotency} from "@bufbuild/protobuf"
import {ConnectError, Interceptor} from "@connectrpc/connect"

export const BOOT_HEADER = "X-Server-Boot"

// The server draws a new boot at each start, and the page may have been built for the one before.
export class Restarts {
    private boot: string | undefined
    private restarted = false

    constructor(private readonly reload: () => void) {}

    public heard(boot: string | null): void {
        if (!boot || this.restarted) return
        if (this.boot === undefined) {
            this.boot = boot
            return
        }
        if (boot === this.boot) return

        this.restarted = true
        this.reload()
    }
}

export function restartInterceptor(restarts: Restarts): Interceptor {
    return (next) => async (req) => {
        // A GET may come from the browser's cache, under a boot that is gone.
        const cacheable = !req.stream && req.method.idempotency === MethodIdempotency.NoSideEffects
        try {
            const res = await next(req)
            if (!cacheable) restarts.heard(res.header.get(BOOT_HEADER))
            return res
        } catch (e) {
            if (!cacheable && e instanceof ConnectError) restarts.heard(e.metadata.get(BOOT_HEADER))
            throw e
        }
    }
}
