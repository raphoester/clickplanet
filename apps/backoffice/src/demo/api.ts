import {createPromiseClient, createRouterTransport} from "@connectrpc/connect"
import {AdminService as AlertsAdmin} from "../gen/grpc/alerts/v1/admin_connect.ts"
import {AdminService as AntibotAdmin} from "../gen/grpc/antibot/v1/admin_connect.ts"
import {AdminService as AuthAdmin} from "../gen/grpc/auth/v1/admin_connect.ts"
import {AdminService as BonusesAdmin} from "../gen/grpc/bonuses/v1/admin_connect.ts"
import {AdminService as ChatAdmin} from "../gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as ClipsAdmin} from "../gen/grpc/clips/v1/admin_connect.ts"
import {AdminService as PlanetAdmin} from "../gen/grpc/planet/v1/admin_connect.ts"
import {AdminService as PlayerAdmin} from "../gen/grpc/player/v1/admin_connect.ts"
import {AdminService as ProgressionAdmin} from "../gen/grpc/progression/v1/admin_connect.ts"
import {AdminService as GamemasterAdmin} from "../gen/grpc/gamemaster/v1/admin_connect.ts"
import {AdminService as MissionsAdmin} from "../gen/grpc/missions/v1/admin_connect.ts"
import {AdminService as PromoAdmin} from "../gen/grpc/promo/v1/admin_connect.ts"
import {AdminService as PublishingAdmin} from "../gen/grpc/publishing/v1/admin_connect.ts"
import {AdminService as SupportAdmin} from "../gen/grpc/support/v1/admin_connect.ts"
import {DemoAdmin, OPERATOR} from "./demoAdmin.ts"

// `npm run demo` serves this module in place of ../api.ts (vite.config.ts): every call answered in the page.
const admin = new DemoAdmin()

const transport = createRouterTransport(router => {
    router.service(AlertsAdmin, admin.demoAlerts)
    router.service(AntibotAdmin, admin.demoAntibot)
    router.service(AuthAdmin, {whoAmI: () => ({accountId: OPERATOR}), signOut: () => ({})})
    router.service(BonusesAdmin, admin.bonuses)
    router.service(ChatAdmin, admin)
    router.service(ClipsAdmin, admin)
    router.service(PlanetAdmin, admin)
    router.service(PlayerAdmin, admin)
    router.service(ProgressionAdmin, admin.demoProgression)
    router.service(GamemasterAdmin, admin.gamemaster)
    router.service(MissionsAdmin, admin.missions)
    router.service(SupportAdmin, admin)
    router.service(PromoAdmin, admin.demoPromo)
    router.service(PublishingAdmin, admin)
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

export const LONG_CALL_MS = 120_000

export function putFile(url: string, file: Blob, _contentType: string, onSent: (bytes: number) => void): Promise<void> {
    return admin.receive(url, file, onSent)
}
