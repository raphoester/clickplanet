import {PromiseClient} from "@connectrpc/connect"
import {chatAdmin, planetAdmin, playerAdmin, supportAdmin} from "./api.ts"
import {AdminService as ChatAdminService} from "./gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as PlanetAdminService} from "./gen/grpc/planet/v1/admin_connect.ts"
import {AdminService as PlayerAdminService} from "./gen/grpc/player/v1/admin_connect.ts"
import {AdminService as SupportAdminService} from "./gen/grpc/support/v1/admin_connect.ts"

// Each page takes the services it calls as these, so a test hands it FakeAdmin (fakeAdmin.ts) instead.
export type ChatAdmin = Pick<PromiseClient<typeof ChatAdminService>, "getMessage" | "listMessages" | "removeMessage" | "mute">
export type PlanetAdmin = PromiseClient<typeof PlanetAdminService>
export type PlayerAdmin = PromiseClient<typeof PlayerAdminService>
export type SupportAdmin = Pick<PromiseClient<typeof SupportAdminService>, "listNotes" | "decideReport">

export type Clients = {chat: ChatAdmin, planet: PlanetAdmin, player: PlayerAdmin, support: SupportAdmin}

export const CLIENTS: Clients = {chat: chatAdmin, planet: planetAdmin, player: playerAdmin, support: supportAdmin}
