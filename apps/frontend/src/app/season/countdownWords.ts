import {CountdownKind} from "../../domain/seasonClock.ts"
import {Key} from "../../i18n/translator.ts"

export type CountdownWords = {label: Key, line: Key, tag: Key}

export const COUNTDOWN_WORDS: Record<CountdownKind, CountdownWords> = {
    today: {label: "season.todayEndsIn", line: "season.todayEndsInTime", tag: "season.todayTag"},
    battle: {label: "battle.nextIn", line: "battle.nextInTime", tag: "battle.tag"},
    final: {label: "finale.in", line: "finale.inTime", tag: "finale.tag"},
    season: {label: "season.endsIn", line: "season.endsInTime", tag: "season.tag"},
}
