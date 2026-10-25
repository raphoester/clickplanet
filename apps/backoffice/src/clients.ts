import {PromiseClient} from "@connectrpc/connect"
import {chatAdmin, clipsAdmin, metricsAdmin, planetAdmin, playerAdmin, seasonsAdmin, supportAdmin} from "./api.ts"
import {AdminService as ChatAdminService} from "./gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as ClipsAdminService} from "./gen/grpc/clips/v1/admin_connect.ts"
import {AdminService as MetricsAdminService} from "./gen/grpc/metrics/v1/admin_connect.ts"
import {AdminService as PlanetAdminService} from "./gen/grpc/planet/v1/admin_connect.ts"
import {AdminService as PlayerAdminService} from "./gen/grpc/player/v1/admin_connect.ts"
import {AdminService as SeasonsAdminService} from "./gen/grpc/seasons/v1/admin_connect.ts"
import {AdminService as SupportAdminService} from "./gen/grpc/support/v1/admin_connect.ts"

// Each page takes the services it calls as these, so a test hands it FakeAdmin (fakeAdmin.ts) instead.
export type ChatAdmin = Pick<PromiseClient<typeof ChatAdminService>, "getMessage" | "listMessages" | "removeMessage" | "mute">
export type PlanetAdmin = PromiseClient<typeof PlanetAdminService>
export type PlayerAdmin = PromiseClient<typeof PlayerAdminService>
export type SupportAdmin = Pick<PromiseClient<typeof SupportAdminService>, "listNotes" | "listReports" | "decideReport">

// The page reads; the clips job keeps (KeepSnapshot), over ssh.
export type ClipsAdmin = Pick<PromiseClient<typeof ClipsAdminService>, "listSnapshots" | "listRows">
export type MetricsAdmin = PromiseClient<typeof MetricsAdminService>
export type SeasonsAdmin = PromiseClient<typeof SeasonsAdminService>

export type Clients = {
    chat: ChatAdmin, planet: PlanetAdmin, player: PlayerAdmin, support: SupportAdmin, clips: ClipsAdmin,
    metrics: MetricsAdmin, seasons: SeasonsAdmin,
}

export const CLIENTS: Clients = {
    chat: chatAdmin, planet: planetAdmin, player: playerAdmin, support: supportAdmin, clips: clipsAdmin,
    metrics: metricsAdmin, seasons: seasonsAdmin,
}
