import {LONG_CALL_MS} from "../api.ts"
import {PlayerAdmin} from "../clients.ts"

export type Reconciled = {granted: number, revoked: number}
export type Named = {named: number}

// Both page through every player with stats, so both take the long timeout.

// Every account's titles made what the rules give it, no more. A second run changes nothing.
export async function reconcileTitles(player: PlayerAdmin): Promise<Reconciled> {
    const answer = await player.reconcileTitles({}, {timeoutMs: LONG_CALL_MS})
    return {granted: answer.granted, revoked: answer.revoked}
}

// A generated name for every signed-in account that has none. A name, chosen or given, is never replaced.
export async function nameAccounts(player: PlayerAdmin): Promise<Named> {
    const answer = await player.nameAccounts({}, {timeoutMs: LONG_CALL_MS})
    return {named: answer.named}
}
