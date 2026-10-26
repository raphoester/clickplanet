import {createPromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {AdminService as AuthAdmin} from "./gen/grpc/auth/v1/admin_connect.ts"
import {AdminService as ChatAdmin} from "./gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as ClipsAdmin} from "./gen/grpc/clips/v1/admin_connect.ts"
import {AdminService as PlanetAdmin} from "./gen/grpc/planet/v1/admin_connect.ts"
import {AdminService as PlayerAdmin} from "./gen/grpc/player/v1/admin_connect.ts"
import {AdminService as SupportAdmin} from "./gen/grpc/support/v1/admin_connect.ts"

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "https://api.clickplanet.lol"

const transport = createConnectTransport({
    baseUrl: BASE_URL,
    // The cp_admin cookie is the operator's only credential: without it every call is unauthenticated.
    fetch: (input, init) => globalThis.fetch(input, {...init, credentials: "include"}),
    defaultTimeoutMs: 10_000,
    // A page of the ledger took the server eight times as long to write as JSON.
    useBinaryFormat: true,
})

export const authAdmin = createPromiseClient(AuthAdmin, transport)
export const chatAdmin = createPromiseClient(ChatAdmin, transport)
export const clipsAdmin = createPromiseClient(ClipsAdmin, transport)
export const planetAdmin = createPromiseClient(PlanetAdmin, transport)
export const playerAdmin = createPromiseClient(PlayerAdmin, transport)
export const supportAdmin = createPromiseClient(SupportAdmin, transport)

// For the calls that walk the whole map or every player: a revert, a reassign, a paint, a replay, a reconciliation.
export const LONG_CALL_MS = 120_000
