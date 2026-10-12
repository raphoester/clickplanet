import {CountryOrder} from "./race.ts";

export const HOLD_MS = 15_000

export type AnthemPick = {
    playing: string | undefined
    order: CountryOrder | undefined
    challenger: {code: string, since: number} | undefined
}

export const NO_PICK: AnthemPick = {playing: undefined, order: undefined, challenger: undefined}

export function followLeader(pick: AnthemPick, leader: string | undefined, order: CountryOrder, now: number): AnthemPick {
    if (leader === undefined) return pick

    if (pick.playing === undefined || pick.order !== order) return {playing: leader, order, challenger: undefined}

    if (leader === pick.playing) {
        return pick.challenger === undefined ? pick : {...pick, challenger: undefined}
    }

    if (pick.challenger?.code !== leader) {
        return {...pick, challenger: {code: leader, since: now}}
    }

    if (now - pick.challenger.since >= HOLD_MS) return {playing: leader, order, challenger: undefined}
    return pick
}
