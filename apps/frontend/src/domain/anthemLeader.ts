/**
 * Which country's anthem plays. It is the leader's, but a new leader has to
 * hold first place for `HOLD_MS` before the music follows: two countries neck
 * and neck swap the lead every few seconds, and restarting an anthem each time
 * is noise, not music.
 */

export const HOLD_MS = 15_000

export type AnthemPick = {
    /** The country whose anthem plays. Undefined until the board has a leader. */
    playing: string | undefined
    /** A leader waiting out its hold, and when it took first place. */
    challenger: {code: string, since: number} | undefined
}

export const NO_PICK: AnthemPick = {playing: undefined, challenger: undefined}

/**
 * Folds the current leader into the pick. Returns `pick` itself when nothing
 * changed, so a caller can skip a render. An empty board keeps what plays.
 */
export function followLeader(pick: AnthemPick, leader: string | undefined, now: number): AnthemPick {
    if (leader === undefined) return pick

    // The first leader plays at once: there is nothing to interrupt.
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
