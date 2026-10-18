import {createPromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {AdminService as AuthAdmin} from "./gen/grpc/auth/v1/admin_connect.ts"
import {AdminService as SupportAdmin} from "./gen/grpc/support/v1/admin_connect.ts"

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "https://api.clickplanet.lol"

const transport = createConnectTransport({
    baseUrl: BASE_URL,
    // The game's cp_sid cookie is the operator's only credential: without it every call is unauthenticated.
    fetch: (input, init) => globalThis.fetch(input, {...init, credentials: "include"}),
    defaultTimeoutMs: 10_000,
})

export const authAdmin = createPromiseClient(AuthAdmin, transport)
export const supportAdmin = createPromiseClient(SupportAdmin, transport)
