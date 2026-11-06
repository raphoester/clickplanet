import {LONG_CALL_MS} from "../api.ts"
import {AlertsAdmin, ProgressionAdmin} from "../clients.ts"

export type Reconciled = {granted: number, revoked: number}

// Every account's titles made what the rules give it, no more. A second run changes nothing.
// It pages through every player with stats, so it takes the long timeout.
export async function reconcileTitles(progression: ProgressionAdmin): Promise<Reconciled> {
    const answer = await progression.reconcileTitles({}, {timeoutMs: LONG_CALL_MS})
    return {granted: answer.granted, revoked: answer.revoked}
}

export type MembersLinked = {linked: number, named: number, outside: number, failed: number}

// Links every account signed in with Discord whose user is in the server already, with its flag's role, and gives every
// linked one its username as its nickname. A second run links and renames nobody twice. It asks Discord about each
// account in turn, so it takes the long timeout.
export async function linkMembers(alerts: AlertsAdmin): Promise<MembersLinked> {
    const answer = await alerts.linkMembers({}, {timeoutMs: LONG_CALL_MS})
    return {linked: answer.linked, named: answer.named, outside: answer.outside, failed: answer.failed}
}
