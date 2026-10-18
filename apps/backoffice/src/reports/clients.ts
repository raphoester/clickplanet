import {chatAdmin, planetAdmin, supportAdmin} from "../api.ts"
import {ChatAdmin, PlanetAdmin, SupportAdmin} from "./actions.ts"

export type Clients = {chat: ChatAdmin, planet: PlanetAdmin, support: SupportAdmin}

export const CLIENTS: Clients = {chat: chatAdmin, planet: planetAdmin, support: supportAdmin}
