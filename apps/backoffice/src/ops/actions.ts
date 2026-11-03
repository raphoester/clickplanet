import {LONG_CALL_MS} from "../api.ts"
import {AlertsAdmin, PlayerAdmin} from "../clients.ts"

export type Reconciled = {granted: number, revoked: number}

// Every account's titles made what the rules give it, no more. A second run changes nothing.
// It pages through every player with stats, so it takes the long timeout.
export async function reconcileTitles(player: PlayerAdmin): Promise<Reconciled> {
    const answer = await player.reconcileTitles({}, {timeoutMs: LONG_CALL_MS})
    return {granted: answer.granted, revoked: answer.revoked}
}

export type MembersLinked = {linked: number, outside: number, failed: number}

// Links every account signed in with Discord whose user is in the server already, with its flag's role. A second run
// links nobody twice. It asks Discord about each account in turn, so it takes the long timeout.
export async function linkMembers(alerts: AlertsAdmin): Promise<MembersLinked> {
    const answer = await alerts.linkMembers({}, {timeoutMs: LONG_CALL_MS})
    return {linked: answer.linked, outside: answer.outside, failed: answer.failed}
}
