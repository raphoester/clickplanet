import {createContext, useContext} from "react"
import {Player} from "./gen/grpc/planet/v1/admin_pb.ts"
import {Caller} from "./players/caller.ts"

export type Section = "support" | "players" | "map" | "ops"

// A player to open on the Players page: who, and what the page that opens it already knows of it.
export type Opened = {caller: Caller, name?: string, row?: Player}

// Opens a player on the Players page from another page, as a report does. Unset outside App, as in a test of one page.
export const OpenPlayer = createContext<((opened: Opened) => void) | undefined>(undefined)

export function useOpenPlayer(): ((opened: Opened) => void) | undefined {
    return useContext(OpenPlayer)
}
