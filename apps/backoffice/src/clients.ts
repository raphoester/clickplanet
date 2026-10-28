import {PromiseClient} from "@connectrpc/connect"
import {antibotAdmin, chatAdmin, clipsAdmin, gamemasterAdmin, planetAdmin, playerAdmin, supportAdmin} from "./api.ts"
import {AdminService as AntibotAdminService} from "./gen/grpc/antibot/v1/admin_connect.ts"
import {AdminService as ChatAdminService} from "./gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as ClipsAdminService} from "./gen/grpc/clips/v1/admin_connect.ts"
import {AdminService as PlanetAdminService} from "./gen/grpc/planet/v1/admin_connect.ts"
import {AdminService as PlayerAdminService} from "./gen/grpc/player/v1/admin_connect.ts"
import {AdminService as GamemasterAdminService} from "./gen/grpc/gamemaster/v1/admin_connect.ts"
import {AdminService as SupportAdminService} from "./gen/grpc/support/v1/admin_connect.ts"

// Each page takes the services it calls as these, so a test hands it FakeAdmin (fakeAdmin.ts) instead.
export type ChatAdmin = Pick<PromiseClient<typeof ChatAdminService>, "getMessage" | "listMessages" | "removeMessage" | "mute">
export type AntibotAdmin = PromiseClient<typeof AntibotAdminService>
// The bans moved to the antibot: planet.v1 keeps the three calls only until this page has called antibot.v1 for a deploy.
export type PlanetAdmin = Omit<PromiseClient<typeof PlanetAdminService>, "banPlayer" | "unbanPlayer" | "inspectPlayer">
export type PlayerAdmin = PromiseClient<typeof PlayerAdminService>
export type GamemasterAdmin = PromiseClient<typeof GamemasterAdminService>
export type SupportAdmin = Pick<PromiseClient<typeof SupportAdminService>, "listNotes" | "listReports" | "decideReport">

// The page reads; the clips job keeps (KeepSnapshot), over ssh.
export type ClipsAdmin = Pick<PromiseClient<typeof ClipsAdminService>, "listSnapshots" | "listRows">

export type Clients = {
    antibot: AntibotAdmin, chat: ChatAdmin, planet: PlanetAdmin, player: PlayerAdmin, gamemaster: GamemasterAdmin, support: SupportAdmin,
    clips: ClipsAdmin,
}

export const CLIENTS: Clients = {
    antibot: antibotAdmin, chat: chatAdmin, planet: planetAdmin, player: playerAdmin, gamemaster: gamemasterAdmin, support: supportAdmin,
    clips: clipsAdmin,
}
