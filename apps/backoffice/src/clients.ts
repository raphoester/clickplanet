import {PromiseClient} from "@connectrpc/connect"
import {
    alertsAdmin, antibotAdmin, chatAdmin, clipsAdmin, gamemasterAdmin, missionsAdmin, planetAdmin, playerAdmin, progressionAdmin,
    promoAdmin, supportAdmin,
} from "./api.ts"
import {AdminService as AlertsAdminService} from "./gen/grpc/alerts/v1/admin_connect.ts"
import {AdminService as AntibotAdminService} from "./gen/grpc/antibot/v1/admin_connect.ts"
import {AdminService as ChatAdminService} from "./gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as ClipsAdminService} from "./gen/grpc/clips/v1/admin_connect.ts"
import {AdminService as PlanetAdminService} from "./gen/grpc/planet/v1/admin_connect.ts"
import {AdminService as PlayerAdminService} from "./gen/grpc/player/v1/admin_connect.ts"
import {AdminService as ProgressionAdminService} from "./gen/grpc/progression/v1/admin_connect.ts"
import {AdminService as GamemasterAdminService} from "./gen/grpc/gamemaster/v1/admin_connect.ts"
import {AdminService as MissionsAdminService} from "./gen/grpc/missions/v1/admin_connect.ts"
import {AdminService as PromoAdminService} from "./gen/grpc/promo/v1/admin_connect.ts"
import {AdminService as SupportAdminService} from "./gen/grpc/support/v1/admin_connect.ts"

// Each page takes the services it calls as these, so a test hands it FakeAdmin (fakeAdmin.ts) instead.
export type ChatAdmin = Pick<PromiseClient<typeof ChatAdminService>, "getMessage" | "listMessages" | "removeMessage" | "mute" | "unmute" | "getMute">
export type AntibotAdmin = PromiseClient<typeof AntibotAdminService>
export type AlertsAdmin = PromiseClient<typeof AlertsAdminService>
// The bans moved to the antibot: planet.v1 keeps the three calls only until this page has called antibot.v1 for a deploy.
export type PlanetAdmin = Omit<PromiseClient<typeof PlanetAdminService>, "banPlayer" | "unbanPlayer" | "inspectPlayer">
export type PlayerAdmin = PromiseClient<typeof PlayerAdminService>
export type ProgressionAdmin = PromiseClient<typeof ProgressionAdminService>
export type GamemasterAdmin = PromiseClient<typeof GamemasterAdminService>
// The numbers missions draws with. The flag days and totals are the connector's reads.
export type MissionsAdmin = Pick<PromiseClient<typeof MissionsAdminService>, "getRules" | "saveRules">
export type SupportAdmin = Pick<PromiseClient<typeof SupportAdminService>, "listNotes" | "listReports" | "decideReport">

// The clips job keeps its snapshots (KeepSnapshot) over ssh; the page reads them, and keeps the videos.
export type ClipsAdmin = Pick<
    PromiseClient<typeof ClipsAdminService>,
    "listSnapshots" | "listRows" | "createVideoUpload" | "keepVideo" | "listVideos" | "removeVideo"
>

// The battle teasers: the background library and each rule set's cards.
export type PromoAdmin = PromiseClient<typeof PromoAdminService>

export type Clients = {
    alerts: AlertsAdmin, antibot: AntibotAdmin, chat: ChatAdmin, planet: PlanetAdmin, player: PlayerAdmin, progression: ProgressionAdmin,
    gamemaster: GamemasterAdmin, missions: MissionsAdmin, support: SupportAdmin, clips: ClipsAdmin, promo: PromoAdmin,
}

export const CLIENTS: Clients = {
    alerts: alertsAdmin, antibot: antibotAdmin, chat: chatAdmin, planet: planetAdmin, player: playerAdmin, progression: progressionAdmin,
    gamemaster: gamemasterAdmin, missions: missionsAdmin, support: supportAdmin, clips: clipsAdmin, promo: promoAdmin,
}
