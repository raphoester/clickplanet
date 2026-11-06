import {useEffect} from "react"
import {StandingsBackend} from "../../backends/standings.ts"
import {TurnBackend} from "../../backends/turn.ts"
import {closedOf, numbersOf} from "../../domain/debrief.ts"
import {NO_SCHEDULE, useSchedule} from "./scheduleStore.ts"

const LONGEST_TIMEOUT_MS = 2 ** 31 - 1

// Follows the schedule into its store, ends the current season on this page's clock, and reads the last debrief's numbers.
export function useScheduleFeed(turn: TurnBackend, standings: StandingsBackend): void {
    useEffect(() => {
        const {hearTurn, hearBattles, hearSeasons, hearDebrief} = useSchedule.getState()
        const stop = turn.follow({onTurn: hearTurn, onBattles: hearBattles, onSeasons: hearSeasons, onDebrief: hearDebrief})
        return () => {
            stop()
            useSchedule.setState(NO_SCHEDULE)
        }
    }, [turn])

    const current = useSchedule((schedule) => schedule.seasons.current)
    useEffect(() => {
        if (!current) return

        let timer: ReturnType<typeof setTimeout> | undefined
        const waitForTheEnd = () => {
            const left = current.endsAt - Date.now()
            if (left <= 0) {
                useSchedule.getState().endSeason(current)
                return
            }
            timer = setTimeout(waitForTheEnd, Math.min(left, LONGEST_TIMEOUT_MS))
        }
        waitForTheEnd()
        return () => clearTimeout(timer)
    }, [current])

    const debrief = useSchedule((schedule) => schedule.debrief)
    useEffect(() => {
        if (!debrief) return

        let stale = false
        const numbers = numbersOf(debrief)
        const reading = numbers.kind === "round"
            ? standings.round(numbers.roundId).then((round) => closedOf(debrief, round))
            : standings.scores(numbers.bookId).then((scores) => closedOf(debrief, undefined, scores))
        reading.then(
            (closed) => {
                if (!stale) useSchedule.getState().readDebrief(debrief, closed)
            },
            (e) => console.error("Could not read the debrief's numbers", e),
        )
        return () => {
            stale = true
        }
    }, [standings, debrief])
}
