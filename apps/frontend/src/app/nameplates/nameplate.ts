import {Nameplate} from "../../backends/nameplates.ts"
import {PlayerLine} from "../../backends/player.ts"
import {Painted} from "../chat/authorStyle.ts"
import {Unnamed} from "./nameplatesStore.ts"

// What a name reads for an account nobody can name: deleted, asked deleted, or a message kept before accounts.
export const DELETED_NAME = "[deleted]"

// Empty until it is read.
export function nameOf(nameplate: Nameplate | Unnamed | undefined): string {
    if (nameplate === null) return DELETED_NAME
    return nameplate?.name ?? ""
}

// Grey for a guest, and for an account not read yet or nobody can name.
export function paintOf(nameplate: Nameplate | Unnamed | undefined): Painted {
    return {color: nameplate?.color, guest: nameplate?.guest ?? false}
}

// The player a card opens on: none for an account nobody can name, nor one not read yet.
export function lineOf(accountId: string, countryCode: string, nameplate: Nameplate | Unnamed | undefined): PlayerLine | undefined {
    if (!nameplate) return undefined
    return {accountId, countryCode, name: nameplate.name, color: nameplate.color, admin: nameplate.admin, guest: nameplate.guest}
}
