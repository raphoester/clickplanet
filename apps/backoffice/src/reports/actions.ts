import {ChatAdmin, SupportAdmin} from "../clients.ts"
import {ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"

export const MAX_COMMENT_LENGTH = 1000

export type Removed = {removedAt: Date}
export type Decided = {outcome: ReportOutcome, comment: string, decidedAt: Date}

export async function removeMessage(chat: ChatAdmin, messageId: string): Promise<Removed> {
    const answer = await chat.removeMessage({messageId})
    return {removedAt: dateOf(answer.removedAt)}
}

export async function decide(
    support: SupportAdmin, reportId: string, outcome: ReportOutcome, comment: string,
): Promise<Decided> {
    const said = comment.trim()
    const answer = await support.decideReport({reportId, outcome, comment: said})
    return {outcome, comment: said, decidedAt: dateOf(answer.decidedAt)}
}

export type Action = "removeMessage" | "decide"

export const MESSAGE_GONE = "This message is no longer kept."
export const REPORT_GONE = "This report is no longer kept."

// What a refusal means for the action it refused, in the operator's words.
export function refusalOf(action: Action, err: unknown): string {
    return wordsOf(err, {notFound: action === "decide" ? REPORT_GONE : MESSAGE_GONE})
}

function dateOf(at: {toDate(): Date} | undefined): Date {
    if (at === undefined) throw new Error("the server answered no time")
    return at.toDate()
}
