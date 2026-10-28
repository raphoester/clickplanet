import {Clients} from "../clients.ts"
import {ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {ban, bannedWords, mute, refusalOf as playerRefusalOf} from "../players/actions.ts"
import {BAN_CHOICES, labelOf, MUTE_CHOICES} from "../players/durations.ts"
import {countOf, outcomeLabel} from "../support/labels.ts"
import {timeOf} from "../components/time.ts"
import {decide, refusalOf, removeMessage} from "./actions.ts"

// What the operator does to the reported account before the report closes. A duration is undefined when it does not
// mute or ban.
export type Verdict = {remove: boolean, muteFor: number | undefined, banFor: number | undefined, comment: string}

export type Target = {accountId: string, messageId: string, who: string}

// What a verdict did, step by step, and the reports it closed.
export type Closed = {who: string, said: string[], outcome: ReportOutcome, comment: string, decidedAt: Date, reportIds: string[]}

// A verdict stopped at a step: what was done before it holds, and the reports not closed yet stay open.
export class Halted extends Error {
    constructor(readonly said: string[], readonly words: string, readonly closed = 0) {
        super(words)
    }
}

// The verdict as the question names it: "remove the message", "mute Ana for 1 hour"…
export function actsOf(verdict: Verdict, who: string): string[] {
    return [
        ...(verdict.remove ? ["remove the message"] : []),
        ...(verdict.muteFor !== undefined ? [`mute ${who} for ${labelOf(MUTE_CHOICES, verdict.muteFor)}`] : []),
        ...(verdict.banFor !== undefined ? [`ban ${who} for ${labelOf(BAN_CHOICES, verdict.banFor)}`] : []),
    ]
}

export function questionOf(acts: string[], reports: number, outcome: ReportOutcome): string {
    const closing = `close ${reports === 1 ? "this report" : `these ${reports} reports`} as ${outcomeLabel(outcome)}`
    const said = [...acts, closing]
    const sentence = said.length === 1 ? said[0] : `${said.slice(0, -1).join(", ")} and ${said[said.length - 1]}`
    return `${sentence.charAt(0).toUpperCase()}${sentence.slice(1)}?`
}

// The verdict's steps one at a time, then each report closed. A comment left empty records what was done.
export async function close(
    clients: Clients, target: Target, verdict: Verdict, outcome: ReportOutcome, reportIds: string[],
): Promise<Closed> {
    const said: string[] = []
    const step = async <T>(run: () => Promise<T>, words: (value: T) => string, refused: (err: unknown) => string) => {
        try {
            said.push(words(await run()))
        } catch (err) {
            throw new Halted(said, refused(err))
        }
    }

    if (verdict.remove) {
        await step(() => removeMessage(clients.chat, target.messageId), () => "Removed the message.",
            err => refusalOf("removeMessage", err))
    }
    const muteFor = verdict.muteFor
    if (muteFor !== undefined) {
        await step(() => mute(clients.chat, target.accountId, muteFor), muted => `Muted until ${timeOf(muted.until)}.`,
            err => playerRefusalOf("mute", err))
    }
    const banFor = verdict.banFor
    if (banFor !== undefined) {
        await step(() => ban(clients.antibot, {accountId: target.accountId}, banFor), bannedWords,
            err => playerRefusalOf("ban", err))
    }

    const comment = verdict.comment.trim() !== "" ? verdict.comment.trim() : pastOf(verdict)
    let decidedAt = new Date()
    for (const [done, reportId] of reportIds.entries()) {
        try {
            decidedAt = (await decide(clients.support, reportId, outcome, comment)).decidedAt
        } catch (err) {
            if (done > 0) said.push(`Closed ${done} of ${countOf(reportIds.length, "report")}.`)
            throw new Halted(said, refusalOf("decide", err), done)
        }
    }
    said.push(`Closed ${countOf(reportIds.length, "report")} as ${outcomeLabel(outcome)}.`)
    return {who: target.who, said, outcome, comment, decidedAt, reportIds}
}

export function haltedWords(err: unknown): string {
    if (!(err instanceof Halted)) return refusalOf("decide", err)
    return [...err.said, err.words, ...(err.closed === 0 ? ["The report is still open."] : [])].join(" ")
}

// What the verdict did, for the report's record when the operator wrote nothing.
function pastOf(verdict: Verdict): string {
    return [
        ...(verdict.remove ? ["Removed the message."] : []),
        ...(verdict.muteFor !== undefined ? [`Muted for ${labelOf(MUTE_CHOICES, verdict.muteFor)}.`] : []),
        ...(verdict.banFor !== undefined ? [`Banned for ${labelOf(BAN_CHOICES, verdict.banFor)}.`] : []),
    ].join(" ")
}
