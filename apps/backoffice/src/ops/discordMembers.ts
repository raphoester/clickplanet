import {countOf} from "../support/labels.ts"
import {MembersLinked} from "./actions.ts"

// What a run of LinkMembers did, in the operator's words.
export function membersLinkedWords({linked, named, outside, failed}: MembersLinked): string {
    if (linked === 0 && named === 0 && outside === 0 && failed === 0) {
        return "Nothing to do: every player signed in with Discord is linked and named already."
    }
    const lines = [linked === 0 ? "Nobody new was in the server." : `Linked ${countOf(linked, "player")}.`]
    if (named > 0) {
        lines.push(`Gave ${countOf(named, "player")} their username as their nickname.`)
    }
    if (outside > 0) {
        lines.push(`${countOf(outside, "player")} not in the server: the game offers them "Join Discord".`)
    }
    if (failed > 0) {
        lines.push(`${countOf(failed, "player")} could not be linked or named: the server's log says why, and a run `
            + "again tries them.")
    }
    return lines.join(" ")
}
