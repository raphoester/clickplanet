import {NameColor} from "../gen/grpc/player/v1/color_pb.ts"

// The most accounts one GetNameplates call may name: the server refuses more.
export const MAX_NAMEPLATES_ASKED = 100

// What the page draws for a player's name, wherever it shows one. A guest is its name alone.
export type Nameplate = {
    name: string
    color: NameColor
    admin: boolean
    guest: boolean
}

export type AccountNameplate = Nameplate & {
    accountId: string
}

export interface NameplatesBackend {
    // Only the accounts the other players can name: an unknown one, one deleted or asked deleted, or one never shown is
    // left out. At most MAX_NAMEPLATES_ASKED ids.
    nameplates(accountIds: readonly string[]): Promise<AccountNameplate[]>
}
