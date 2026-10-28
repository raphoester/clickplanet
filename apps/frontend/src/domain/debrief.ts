import {ClosedRound, Round, Score} from "../backends/standings.ts"
import {Debrief} from "../backends/turn.ts"

// What to read from scoring for a debrief: its round, or the book's scores for a season no round finished.
export type Numbers = {kind: "round", roundId: string} | {kind: "scores", bookId: string}

export function numbersOf(debrief: Debrief): Numbers {
    switch (debrief.kind) {
        case "day":
        case "battle":
            return {kind: "round", roundId: debrief.roundId}
        case "season":
            return debrief.finaleRoundId !== ""
                ? {kind: "round", roundId: debrief.finaleRoundId}
                : {kind: "scores", bookId: debrief.bookId}
    }
}

// The round a debrief tells of; none for a season no round finished.
export function debriefRoundOf(debrief: Debrief | undefined): string {
    if (!debrief) return ""
    switch (debrief.kind) {
        case "day":
        case "battle":
            return debrief.roundId
        case "season":
            return debrief.finaleRoundId
    }
}

// The words are the gamemaster's, the numbers scoring's.
export function closedOf(debrief: Debrief, round: Round | undefined, scores: readonly Score[] = []): ClosedRound {
    const numbers = {
        roundId: round?.roundId ?? debriefRoundOf(debrief),
        endedAt: debrief.endedAt,
        standings: round?.standings ?? [],
        before: round?.before ?? [],
        after: round?.after ?? [...scores],
        times: round?.times ?? 1,
        startedAt: round?.startedAt ?? 0,
        takes: round?.takes ?? [],
        players: round?.players ?? [],
    }
    switch (debrief.kind) {
        case "day":
            return {...numbers, moment: "day", season: debrief.season, number: debrief.number, finale: false, entryId: ""}
        case "battle":
            return {
                ...numbers,
                moment: "battle",
                season: debrief.season,
                number: debrief.number,
                finale: debrief.endsSeason,
                times: round?.times ?? debrief.times,
                entryId: debrief.entryId,
            }
        case "season": {
            const battle = debrief.finaleBattle
            return {
                ...numbers,
                moment: "season",
                season: debrief.season,
                number: battle?.number ?? 0,
                finale: true,
                times: round?.times ?? battle?.times ?? 1,
                entryId: battle?.entryId ?? "",
            }
        }
    }
}

// The season a debrief closed: its book stays the board's until another season runs.
export function endedBookOf(debrief: Debrief | undefined): string | undefined {
    return debrief?.kind === "season" ? debrief.bookId : undefined
}
