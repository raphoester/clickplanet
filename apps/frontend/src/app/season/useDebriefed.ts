import {useEffect, useState} from "react"
import {ClosedRound, StandingsBackend} from "../../backends/standings.ts"
import {Debrief} from "../../backends/turn.ts"
import {closedOf, numbersOf} from "../../domain/debrief.ts"

// The last debrief, once its numbers are read from scoring.
export function useDebriefed(backend: StandingsBackend | undefined, debrief: Debrief | undefined): ClosedRound | undefined {
    const [read, setRead] = useState<{debrief: Debrief, closed: ClosedRound}>()

    useEffect(() => {
        if (!backend || !debrief) return

        let stale = false
        const numbers = numbersOf(debrief)
        const reading = numbers.kind === "round"
            ? backend.round(numbers.roundId).then((round) => closedOf(debrief, round))
            : backend.scores(numbers.bookId).then((scores) => closedOf(debrief, undefined, scores))
        reading.then(
            (closed) => {
                if (!stale) setRead({debrief, closed})
            },
            (e) => console.error("Could not read the debrief's numbers", e),
        )
        return () => {
            stale = true
        }
    }, [backend, debrief])

    return read && read.debrief === debrief ? read.closed : undefined
}
