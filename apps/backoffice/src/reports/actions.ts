import {Duration} from "@bufbuild/protobuf"
import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {AdminService as ChatAdminService} from "../gen/grpc/chat/v1/admin_connect.ts"
import {AdminService as PlanetAdminService} from "../gen/grpc/planet/v1/admin_connect.ts"
import {InspectPlayerResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {AdminService as SupportAdminService} from "../gen/grpc/support/v1/admin_connect.ts"
import {ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"

export type ChatAdmin = Pick<PromiseClient<typeof ChatAdminService>, "getMessage" | "removeMessage" | "mute">
export type PlanetAdmin = Pick<PromiseClient<typeof PlanetAdminService>, "banPlayer" | "inspectPlayer">
export type SupportAdmin = Pick<PromiseClient<typeof SupportAdminService>, "decideReport">

// The server's own default for each: an hour of mute, and the ban ladder's next step.
export const DEFAULT_MUTE_SECONDS = 60 * 60
export const LADDER = 0

export const MAX_COMMENT_LENGTH = 1000

export type Removed = {removedAt: Date}
export type Muted = {until: Date}
export type Banned = {until: Date, offence: number, enforced: boolean}
export type Decided = {outcome: ReportOutcome, comment: string, decidedAt: Date}

export async function removeMessage(chat: ChatAdmin, messageId: string): Promise<Removed> {
    const answer = await chat.removeMessage({messageId})
    return {removedAt: dateOf(answer.removedAt)}
}

export async function mute(chat: ChatAdmin, accountId: string, seconds: number): Promise<Muted> {
    const answer = await chat.mute({accountId, duration: new Duration({seconds: BigInt(seconds)})})
    return {until: dateOf(answer.mutedUntil)}
}

export async function ban(planet: PlanetAdmin, accountId: string, seconds: number): Promise<Banned> {
    const duration = seconds === LADDER ? undefined : new Duration({seconds: BigInt(seconds)})
    const answer = await planet.banPlayer({accountId, duration})
    return {until: dateOf(answer.bannedUntil), offence: answer.offence, enforced: answer.enforced}
}

export async function inspect(planet: PlanetAdmin, accountId: string): Promise<InspectPlayerResponse> {
    return planet.inspectPlayer({accountId})
}

export async function decide(
    support: SupportAdmin, reportId: string, outcome: ReportOutcome, comment: string,
): Promise<Decided> {
    const said = comment.trim()
    const answer = await support.decideReport({reportId, outcome, comment: said})
    return {outcome, comment: said, decidedAt: dateOf(answer.decidedAt)}
}

export type Action = "removeMessage" | "mute" | "ban" | "inspect" | "decide"

export const MESSAGE_GONE = "This message is no longer kept."
export const REPORT_GONE = "This report is no longer kept."

// What a refusal means for the action it refused, in the operator's words.
export function refusalOf(action: Action, err: unknown): string {
    const error = ConnectError.from(err)

    switch (error.code) {
        case Code.NotFound:
            return action === "decide" ? REPORT_GONE : MESSAGE_GONE
        case Code.FailedPrecondition:
            return "The antibot is off on this server, so it cannot ban or inspect."
        case Code.InvalidArgument:
            return `The server refused it: ${error.rawMessage}`
        case Code.Unauthenticated:
            return "Your session ended. Reload the page and sign in again."
        case Code.PermissionDenied:
            return "This account is not an operator."
    }
    return `The call failed: ${error.rawMessage}`
}

function dateOf(at: {toDate(): Date} | undefined): Date {
    if (at === undefined) throw new Error("the server answered no time")
    return at.toDate()
}
