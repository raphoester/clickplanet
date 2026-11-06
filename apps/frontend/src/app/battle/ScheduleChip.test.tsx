// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import {PLAIN_TURN} from "../../backends/turn.ts"
import {SEASON_ZERO} from "../../backends/fakeTurnBackend.ts"
import {closedOf} from "../../domain/debrief.ts"
import ScheduleChip from "./ScheduleChip.tsx"
import {NO_SCHEDULE, useSchedule} from "./scheduleStore.ts"

const HOUR = 3_600_000
const season = {...SEASON_ZERO, number: 1, startsAt: Date.now() - HOUR, endsAt: Date.now() + 100 * HOUR}
const battle = {entryId: "tonight", since: Date.now() - HOUR, endsAt: Date.now() + HOUR, description: "", roundId: "tonight"}
const seasonZeroClosed = closedOf(
    {kind: "season", season: 0, bookId: "season-0", endedAt: Date.now() - HOUR, finaleRoundId: "season-0/finale"},
    undefined,
    [{rank: 1, countryCode: "de", points: 90, roundsWon: 4}],
)

afterEach(() => {
    cleanup()
    useSchedule.setState(NO_SCHEDULE)
})

describe("ScheduleChip", () => {
    it("shows the battle that runs before anything else", () => {
        useSchedule.setState({turn: PLAIN_TURN, battle, seasons: {current: season, ended: []}, closed: seasonZeroClosed})
        render(<ScheduleChip compact={false}/>)

        expect(screen.getByRole("button", {name: /^Battle ends in/})).toBeDefined()
        expect(screen.queryByRole("region", {name: "Germany wins Season 0"})).toBeNull()
    })

    it("shows the last season's champion while no battle runs and no later season does", () => {
        useSchedule.setState({turn: PLAIN_TURN, seasons: {next: season, ended: []}, closed: seasonZeroClosed})
        render(<ScheduleChip compact={false}/>)

        expect(screen.getByRole("region", {name: "Germany wins Season 0"})).toBeDefined()
    })

    it("shows the season's clocks once a later season runs", () => {
        useSchedule.setState({turn: PLAIN_TURN, seasons: {current: season, ended: []}, closed: seasonZeroClosed})
        render(<ScheduleChip compact={false}/>)

        expect(screen.getByRole("region", {name: "Season 1"})).toBeDefined()
        expect(screen.queryByRole("region", {name: "Germany wins Season 0"})).toBeNull()
    })

    it("shows nothing between two seasons with no champion", () => {
        useSchedule.setState({turn: PLAIN_TURN, seasons: {ended: []}})
        const {container} = render(<ScheduleChip compact/>)

        expect(container.innerHTML).toBe("")
    })
})
