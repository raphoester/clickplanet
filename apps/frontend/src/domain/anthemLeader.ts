export const HOLD_MS = 15_000

export type AnthemPick = {
    playing: string | undefined
    challenger: {code: string, since: number} | undefined
}

export const NO_PICK: AnthemPick = {playing: undefined, challenger: undefined}

export function followLeader(pick: AnthemPick, leader: string | undefined, now: number): AnthemPick {
    if (leader === undefined) return pick

    if (pick.playing === undefined) return {playing: leader, challenger: undefined}

    if (leader === pick.playing) {
        return pick.challenger === undefined ? pick : {playing: pick.playing, challenger: undefined}
    }

    if (pick.challenger?.code !== leader) {
        return {playing: pick.playing, challenger: {code: leader, since: now}}
    }

    if (now - pick.challenger.since >= HOLD_MS) return {playing: leader, challenger: undefined}
    return pick
}
