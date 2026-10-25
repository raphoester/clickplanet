import {LONG_CALL_MS} from "../api.ts"
import {PlayerAdmin} from "../clients.ts"

export type Reconciled = {granted: number, revoked: number}

// Every account's titles made what the rules give it, no more. A second run changes nothing.
// It pages through every player with stats, so it takes the long timeout.
export async function reconcileTitles(player: PlayerAdmin): Promise<Reconciled> {
    const answer = await player.reconcileTitles({}, {timeoutMs: LONG_CALL_MS})
    return {granted: answer.granted, revoked: answer.revoked}
}

// Every account's bombs dropped and tiles fortified set to what planet's ledger holds. A second run changes nothing.
export async function recountStats(player: PlayerAdmin): Promise<number> {
    const answer = await player.recountStats({}, {timeoutMs: LONG_CALL_MS})
    return answer.recounted
}
