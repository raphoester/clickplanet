import {createPromiseClient, createRouterTransport} from "@connectrpc/connect"
import {AdminService as AntibotAdmin} from "../gen/grpc/antibot/v1/admin_connect.ts"
import {AdminService as AuthAdmin} from "../gen/grpc/auth/v1/admin_connect.ts"
import {AdminService as ChatAdmin} from "../gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as ClipsAdmin} from "../gen/grpc/clips/v1/admin_connect.ts"
import {AdminService as PlanetAdmin} from "../gen/grpc/planet/v1/admin_connect.ts"
import {AdminService as PlayerAdmin} from "../gen/grpc/player/v1/admin_connect.ts"
import {AdminService as GamemasterAdmin} from "../gen/grpc/gamemaster/v1/admin_connect.ts"
import {AdminService as SupportAdmin} from "../gen/grpc/support/v1/admin_connect.ts"
import {DemoAdmin, OPERATOR} from "./demoAdmin.ts"

// `npm run demo` serves this module in place of ../api.ts (vite.config.ts): every call answered in the page.
const admin = new DemoAdmin()

const transport = createRouterTransport(router => {
    router.service(AntibotAdmin, admin.demoAntibot)
    router.service(AuthAdmin, {whoAmI: () => ({accountId: OPERATOR}), signOut: () => ({})})
    router.service(ChatAdmin, admin)
    router.service(ClipsAdmin, admin)
    router.service(PlanetAdmin, admin)
    router.service(PlayerAdmin, admin)
    router.service(GamemasterAdmin, admin.gamemaster)
    router.service(SupportAdmin, admin)
})

export const antibotAdmin = createPromiseClient(AntibotAdmin, transport)
export const authAdmin = createPromiseClient(AuthAdmin, transport)
export const chatAdmin = createPromiseClient(ChatAdmin, transport)
export const clipsAdmin = createPromiseClient(ClipsAdmin, transport)
export const planetAdmin = createPromiseClient(PlanetAdmin, transport)
export const playerAdmin = createPromiseClient(PlayerAdmin, transport)
export const gamemasterAdmin = createPromiseClient(GamemasterAdmin, transport)
export const supportAdmin = createPromiseClient(SupportAdmin, transport)

export const LONG_CALL_MS = 120_000
