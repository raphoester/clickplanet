export type Timeline = {
    seconds: number
    ending: number
}

const ENDING_SECONDS = 2.5

const SHORTEST_SECONDS = 14

const LONGEST_SECONDS = 22

// A bomb holds the clip still while it falls and goes off: about a tenth of it for one bomb.
const BOMB_SHARE = 0.12

// When something happened, and how much of the clip it is worth: a tile changing hands is 1.
export type Moment = {at: number, weight: number}

export type Pace = {
    timeAt(share: number): number
    shareOf(at: number): number
    heldOn(at: number): {from: number, to: number}
}

// The more tiles change hands, the longer the clip, within what a feed holds a viewer for.
export function timelineOf(action: number, seconds?: number): Timeline {
    const chosen = seconds ?? Math.round(2 * (8 + 3 * Math.log10(Math.max(1, action)))) / 2
    return {seconds: Math.min(LONGEST_SECONDS, Math.max(SHORTEST_SECONDS, chosen)), ending: ENDING_SECONDS}
}

// The share of the replay played after so many seconds of video: all of it once the ending starts.
export function playedAt({seconds, ending}: Timeline, at: number): number {
    const playing = seconds - ending
    return playing <= 0 ? 1 : Math.min(1, Math.max(0, at / playing))
}

export function momentsOf(changes: readonly number[], bombs: readonly number[]): Moment[] {
    const bomb = Math.max(1, changes.length * BOMB_SHARE)
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
