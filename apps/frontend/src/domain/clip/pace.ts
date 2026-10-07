export type Timeline = {
    seconds: number
    ending: number
}

const ENDING_SECONDS = 2.5

// A clip lasts as long as its camera has somewhere to go: a fight in one place is over in a few seconds, however
// long it went on, and only one that crosses the map gets long.
const SHORTEST_PLAY_SECONDS = 5

const SECONDS_PER_SCREEN = 1.2

const LONGEST_PLAY_SECONDS = 15

// A bomb gets the seconds it takes to fall and go off.
const BOMB_SECONDS = 2

const LONGEST_SECONDS = 22

// Bombs never take more than this much of the clip between them.
const MOST_FOR_BOMBS = 0.6

// When something happened, and how much of the clip it is worth: a tile changing hands is 1.
export type Moment = {at: number, weight: number}

export type Pace = {
    timeAt(share: number): number
    shareOf(at: number): number
    heldOn(at: number): {from: number, to: number}
}

// screens: how far the action moves up close, in screen heights.
export function timelineOf(screens: number, bombs: number, seconds?: number): Timeline {
    const play = Math.min(LONGEST_PLAY_SECONDS, SHORTEST_PLAY_SECONDS + SECONDS_PER_SCREEN * screens) + BOMB_SECONDS * bombs
    const chosen = seconds ?? Math.round(2 * (play + ENDING_SECONDS)) / 2
    return {seconds: Math.min(LONGEST_SECONDS, chosen), ending: ENDING_SECONDS}
}

// The share of the clip's play one bomb holds still for.
export function bombShareOf({seconds, ending}: Timeline): number {
    return BOMB_SECONDS / Math.max(seconds - ending, BOMB_SECONDS)
}

// The share of the replay played after so many seconds of video: all of it once the ending starts.
export function playedAt({seconds, ending}: Timeline, at: number): number {
    const playing = seconds - ending
    return playing <= 0 ? 1 : Math.min(1, Math.max(0, at / playing))
}

// bombShare: the share of the clip each bomb holds still for.
export function momentsOf(changes: readonly number[], bombs: readonly number[], bombShare: number): Moment[] {
    const share = Math.min(bombShare, MOST_FOR_BOMBS / Math.max(1, bombs.length))
    const bomb = Math.max(1, changes.length * share / (1 - bombs.length * share))
    return [...changes.map((at) => ({at, weight: 1})), ...bombs.map((at) => ({at, weight: bomb}))]
}

// The clip is spent on what happens and nothing else: a quiet hour takes no time at all.
export function paceOf(moments: readonly Moment[], since: number, until: number): Pace {
    const sorted = moments.filter(({at}) => at >= since && at <= until).sort((a, b) => a.at - b.at)
    if (sorted.length === 0) {
        const span = Math.max(1, until - since)
        const shareOf = (at: number) => (Math.min(until, Math.max(since, at)) - since) / span
        return {
            shareOf,
            timeAt: (share) => since + Math.min(1, Math.max(0, share)) * span,
            heldOn: (at) => ({from: shareOf(at), to: shareOf(at)}),
        }
    }

    const reached: number[] = []
    let total = 0
    for (const {weight} of sorted) reached.push(total += weight)

    // The number of moments at or before at, or strictly before it.
    const countUpTo = (at: number, strictly: boolean) => {
        let low = 0
        let high = sorted.length
        while (low < high) {
            const middle = (low + high) >> 1
            if (strictly ? sorted[middle].at < at : sorted[middle].at <= at) low = middle + 1
            else high = middle
        }
        return low
    }
    const shareAfter = (count: number) => count === 0 ? 0 : reached[count - 1] / total

    return {
        shareOf: (at) => shareAfter(countUpTo(at, false)),
        heldOn: (at) => ({from: shareAfter(countUpTo(at, true)), to: shareAfter(countUpTo(at, false))}),
        timeAt(share) {
            let low = 0
            let high = sorted.length - 1
            while (low < high) {
                const middle = (low + high) >> 1
                if (reached[middle] / total < share) low = middle + 1
                else high = middle
            }
            return sorted[low].at
        },
    }
}
