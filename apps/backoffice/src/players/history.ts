import {ChatAdmin, PlayerAdmin} from "../clients.ts"
import {ModerationAction} from "../gen/grpc/chat/v1/admin_pb.ts"
import {Rename} from "../gen/grpc/player/v1/admin_pb.ts"
import {dateOf, muteWordsOf} from "./actions.ts"
import {Duration} from "@bufbuild/protobuf"
import {MUTE_CHOICES, spanOf} from "./durations.ts"

// Who took an action: Claude, through the connector; the operator, on this page; or a script on the console listener,
// which has no gate and names nobody.
export type Actor = {kind: "app", name: string} | {kind: "operator", accountId: string} | {kind: "console"}

// One line of an account's moderation history: what was done, why, and by whom.
export type Moderated = {
    id: string, at: Date, what: string, reason: string, by: Actor,
    // What a removed message said, kept with its removal past the chat's retention, or its GIF's address.
    said?: {text: string} | {gifUrl: string},
}

// The history keeps 90 days; the server answers 500 at most.
export const HISTORY_LIMIT = 500

// The chat's mutes, unmutes and removals and the player module's renames of one account, newest first.
export async function historyOf(
    chat: ChatAdmin, player: PlayerAdmin, accountId: string, signal?: AbortSignal,
): Promise<Moderated[]> {
    const [moderation, renames] = await Promise.all([
        chat.listModeration({accountId, limit: HISTORY_LIMIT}, {signal}),
        player.listRenames({accountId, limit: HISTORY_LIMIT}, {signal}),
    ])
    return [...moderation.actions.map(moderatedOf), ...renames.renames.map(renamedOf)]
        .sort((a, b) => b.at.getTime() - a.at.getTime())
}

export function actorOf(operatorAccountId: string, client: string): Actor {
    if (client !== "") return {kind: "app", name: client}
    if (operatorAccountId !== "") return {kind: "operator", accountId: operatorAccountId}
    return {kind: "console"}
}

function moderatedOf(action: ModerationAction): Moderated {
    const base = {
        id: `chat-${action.actionId}`, at: dateOf(action.actedAt), reason: action.reason,
        by: actorOf(action.operatorAccountId, action.client),
    }
    switch (action.action.case) {
    case "mute":
        return {...base, what: `${muteWordsOf(action.action.value.kind).muted} for ${lengthOf(action.action.value.duration)}`}
    case "unmute":
        return {...base, what: muteWordsOf(action.action.value.kind).unmuted}
    case "removal": {
        const body = action.action.value.body
        if (body.case === "gif") return {...base, what: "Removed a GIF", said: {gifUrl: body.value.url}}
        return {...base, what: "Removed a message", said: {text: body.value ?? ""}}
    }
    default:
        return {...base, what: "An action this page does not know"}
    }
}

function renamedOf(rename: Rename): Moderated {
    return {
        id: `rename-${rename.renameId}`, at: dateOf(rename.renamedAt), reason: rename.reason,
        what: `Renamed from ${rename.oldName} to ${rename.newName}`,
        by: actorOf(rename.operatorAccountId, rename.client),
    }
}

// "1 hour", as the mute tool offers it, or else "2h 30m".
function lengthOf(duration: Duration | undefined): string {
    const seconds = Number(duration?.seconds ?? 0n)
    return MUTE_CHOICES.find(choice => choice.seconds === seconds)?.label ?? spanOf(duration)
}

// Who, in the page's words.
export function actorWords(by: Actor): string {
    switch (by.kind) {
    case "app":
        return by.name
    case "operator":
        return "Operator"
    case "console":
        return "Console"
    }
}
