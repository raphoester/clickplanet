export type Timeline = {
    seconds: number
    ending: number
}

const ENDING_SECONDS = 2.5

// A clip lasts as long as its camera has somewhere to go: a fight in one place is over in a few seconds, however
// long it went on, and only one that crosses the map gets long. The shortest still has room for the dive in, the
// close-ups, and the pull back out to the map as it is now.
const SHORTEST_PLAY_SECONDS = 6

const SECONDS_PER_SCREEN = 1.2

const LONGEST_PLAY_SECONDS = 15

const LONGEST_SECONDS = 22

export type Pace = {
    timeAt(share: number): number
    shareOf(at: number): number
}

// A day's podium is three flags, their names and their points to read before the call.
export const PODIUM_ENDING_SECONDS = 4

// screens: how far the action moves up close, in screen heights. A bomb takes no time of its own: it goes off while
// the map goes on changing.
export function timelineOf(screens: number, seconds?: number, ending = ENDING_SECONDS): Timeline {
    const play = Math.min(LONGEST_PLAY_SECONDS, SHORTEST_PLAY_SECONDS + SECONDS_PER_SCREEN * screens)
    return timelineFor(play, seconds, ending)
}

// play: how long the replay plays before the call to act, which lasts ending.
export function timelineFor(play: number, seconds?: number, ending = ENDING_SECONDS): Timeline {
    const chosen = seconds ?? Math.round(2 * (play + ending)) / 2
    return {seconds: Math.min(LONGEST_SECONDS, chosen), ending}
}

// The share of the replay played after so many seconds of video: all of it once the ending starts.
export function playedAt({seconds, ending}: Timeline, at: number): number {
    const playing = seconds - ending
    return playing <= 0 ? 1 : Math.min(1, Math.max(0, at / playing))
}

// The clip is spent on what happens and nothing else: a quiet hour takes no time at all. changes: when each tile
// changed hands.
export function paceOf(changes: readonly number[], since: number, until: number): Pace {
    const sorted = changes.filter((at) => at >= since && at <= until).sort((a, b) => a - b)
    if (sorted.length === 0) {
        const span = Math.max(1, until - since)
        return {
            shareOf: (at) => (Math.min(until, Math.max(since, at)) - since) / span,
            timeAt: (share) => since + Math.min(1, Math.max(0, share)) * span,
        }
    }

    // The number of changes at or before at.
    const countUpTo = (at: number) => {
        let low = 0
        let high = sorted.length
        while (low < high) {
            const middle = (low + high) >> 1
            if (sorted[middle] <= at) low = middle + 1
            else high = middle
        }
        return low
    }

    return {
        shareOf: (at) => countUpTo(at) / sorted.length,
        timeAt: (share) => sorted[Math.min(sorted.length - 1, Math.max(0, Math.ceil(share * sorted.length) - 1))],
    }
}

// A saga spends part of its play on the calendar, so a week held reads as a week going by, and the rest on what
// happens, by the tiles that changed. Each change is reached over lead before it rather than at once, so the clock runs
// into an enclosure of a thousand tiles and lands it, and never stands still on it. calendar: the share of the play
// spent evenly on time.
export function blendedPaceOf(changes: readonly number[], since: number, until: number, calendar: number, lead: number): Pace {
    const sorted = changes.filter((at) => at >= since && at <= until).map((at) => at - since).sort((a, b) => a - b)
    const span = Math.max(1, until - since)
    const sums = new Float64Array(sorted.length + 1)
    sorted.forEach((at, i) => sums[i + 1] = sums[i] + at)
    const countUpTo = (at: number) => {
        let low = 0
        let high = sorted.length
        while (low < high) {
            const middle = (low + high) >> 1
            if (sorted[middle] <= at) low = middle + 1
            else high = middle
        }
        return low
    }
    // The changes made by at, and the share of each one coming within lead.
    const reached = (at: number) => {
        const made = countUpTo(at)
        const coming = countUpTo(at + lead) - made
        return made + (coming === 0 ? 0 : (coming * (at + lead) - (sums[made + coming] - sums[made])) / lead)
    }
    const shareOf = (at: number) => {
        const elapsed = Math.min(until, Math.max(since, at)) - since
        const busy = sorted.length === 0 ? elapsed / span : reached(elapsed) / sorted.length
        return calendar * elapsed / span + (1 - calendar) * busy
    }
    return {
        shareOf,
        timeAt: (share) => {
            let low = since
            let high = until
            while (high - low > 1) {
                const middle = Math.floor((low + high) / 2)
                if (shareOf(middle) < share) low = middle
                else high = middle
            }
            return high
        },
    }
}
