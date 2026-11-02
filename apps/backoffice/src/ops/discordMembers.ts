import {countOf} from "../support/labels.ts"
import {MembersLinked} from "./actions.ts"

// What a run of LinkMembers did, in the operator's words.
export function membersLinkedWords({linked, outside, failed}: MembersLinked): string {
    if (linked === 0 && outside === 0 && failed === 0) {
        return "Nothing to do: every player signed in with Discord is linked already."
    }
    const lines = [linked === 0 ? "Nobody new was in the server." : `Linked ${countOf(linked, "player")}.`]
    if (outside > 0) {
        lines.push(`${countOf(outside, "player")} not in the server: the game offers them "Join Discord".`)
    }
    if (failed > 0) {
        lines.push(`${countOf(failed, "player")} could not be checked: run it again, and the server's log says why.`)
    }
    return lines.join(" ")
}
