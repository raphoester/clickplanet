import {createPromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {AdminService as AlertsAdmin} from "./gen/grpc/alerts/v1/admin_connect.ts"
import {AdminService as AntibotAdmin} from "./gen/grpc/antibot/v1/admin_connect.ts"
import {AdminService as AuthAdmin} from "./gen/grpc/auth/v1/admin_connect.ts"
import {AdminService as BonusesAdmin} from "./gen/grpc/bonuses/v1/admin_connect.ts"
import {AdminService as ChatAdmin} from "./gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as ClipsAdmin} from "./gen/grpc/clips/v1/admin_connect.ts"
import {AdminService as PublishingAdmin} from "./gen/grpc/publishing/v1/admin_connect.ts"
import {AdminService as PlanetAdmin} from "./gen/grpc/planet/v1/admin_connect.ts"
import {AdminService as PlayerAdmin} from "./gen/grpc/player/v1/admin_connect.ts"
import {AdminService as ProgressionAdmin} from "./gen/grpc/progression/v1/admin_connect.ts"
import {AdminService as GamemasterAdmin} from "./gen/grpc/gamemaster/v1/admin_connect.ts"
import {AdminService as MissionsAdmin} from "./gen/grpc/missions/v1/admin_connect.ts"
import {AdminService as PromoAdmin} from "./gen/grpc/promo/v1/admin_connect.ts"
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

export const alertsAdmin = createPromiseClient(AlertsAdmin, transport)
export const antibotAdmin = createPromiseClient(AntibotAdmin, transport)
export const authAdmin = createPromiseClient(AuthAdmin, transport)
export const bonusesAdmin = createPromiseClient(BonusesAdmin, transport)
export const chatAdmin = createPromiseClient(ChatAdmin, transport)
export const clipsAdmin = createPromiseClient(ClipsAdmin, transport)
export const planetAdmin = createPromiseClient(PlanetAdmin, transport)
export const playerAdmin = createPromiseClient(PlayerAdmin, transport)
export const progressionAdmin = createPromiseClient(ProgressionAdmin, transport)
export const gamemasterAdmin = createPromiseClient(GamemasterAdmin, transport)
export const missionsAdmin = createPromiseClient(MissionsAdmin, transport)
export const supportAdmin = createPromiseClient(SupportAdmin, transport)
export const promoAdmin = createPromiseClient(PromoAdmin, transport)
export const publishingAdmin = createPromiseClient(PublishingAdmin, transport)

// For the calls that walk the whole map or every player: a revert, a reassign, a paint, a replay, a reconciliation.
export const LONG_CALL_MS = 120_000

// XMLHttpRequest, not fetch: fetch cannot report how much of an upload is sent.
export function putFile(url: string, file: Blob, contentType: string, onSent: (bytes: number) => void): Promise<void> {
    return new Promise((resolve, reject) => {
        const request = new XMLHttpRequest()
        request.open("PUT", url)
        request.setRequestHeader("Content-Type", contentType)
        request.upload.onprogress = event => onSent(event.loaded)
        request.onload = () => {
            if (request.status >= 200 && request.status < 300) resolve()
            else reject(new Error(`the bucket answered ${request.status}${saidOf(request.responseText)}`))
        }
        request.onerror = () => reject(new Error("it did not reach the bucket"))
        request.send(file)
    })
}

function saidOf(answer: string): string {
    const message = /<Message>([^<]*)<\/Message>/.exec(answer)?.[1]
    return message === undefined ? "" : `: ${message}`
}
