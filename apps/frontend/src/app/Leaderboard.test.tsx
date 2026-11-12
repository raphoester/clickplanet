// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {Race} from "../backends/standings.ts"
import Leaderboard from "./Leaderboard.tsx"
import {FIGURES, GUIDE_MS, TAP_HINT_MS} from "./boardFigures.ts"
import {t} from "../i18n/page.ts"
import {useBoardGuide} from "./boardGuideStore.ts"
import {useRace} from "./standings/raceStore.ts"
import {Countries} from "../domain/countries.ts"
import {TileDelta, TileDeltas} from "../domain/tileDeltas.ts"

const entry = (code: string, tiles: number) => ({country: Countries.get(code)!, tiles})

const deltas = (pairs: Record<string, number>): TileDeltas => new Map(
    Object.entries(pairs).map(([code, net]): [string, TileDelta] =>
        [code, {net, beat: 1, at: 0}]))

const leader = () => screen.queryByRole("region", {name: /^First: /})
const rows = () => screen.queryAllByRole("row").slice(1)
const cells = () => rows().map(r => within(r).getAllByRole("cell").map(c => c.textContent))
const headers = (...names: string[]) => {
    expect(screen.getAllByRole("columnheader")).toHaveLength(names.length)
    for (const name of names) expect(screen.getByRole("columnheader", {name})).toBeDefined()
}

beforeEach(() => useBoardGuide.setState({guided: true}))

afterEach(() => {
    cleanup()
    localStorage.clear()
    useRace.setState({race: undefined, heard: undefined})
    vi.useRealTimers()
})

describe("Leaderboard", () => {
    it("frames the first country, then lists the rest in the order it was given", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250), entry("de", 100)]}/>)

        expect(leader()!.getAttribute("aria-label")).toBe("First: France")
        expect(cells()).toEqual([
            ["2", "Japan", "250 25%"],
            ["3", "Germany", "100 10%"],
        ])
    })

    it("gives the first country its share of the map and its tiles", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 123)]}/>)

        expect(within(leader()!).getByText("12.30")).toBeDefined()
        expect(leader()!.querySelector(".leader-frame-tiles")!.textContent).toBe("123 tiles")
    })

    it("writes each country's share of the map beside its tiles, in whole percents", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 123), entry("de", 1)]}/>)
        expect(cells().map((row) => row[2])).toEqual(["123 12%", "1 <1%"])
    })

    it("says a share is small rather than rounding it away to nothing", () => {
        render(<Leaderboard tilesCount={257_948} data={[entry("fr", 1)]}/>)
        expect(screen.getByText("<0.01")).toBeDefined()
    })

    it("still writes a plain zero for a country holding no tile at all", () => {
        render(<Leaderboard tilesCount={257_948} data={[entry("fr", 0)]}/>)
        expect(screen.getByText("0.00")).toBeDefined()
    })

    it("renders no frame and no table when no country holds a tile", () => {
        render(<Leaderboard tilesCount={1000} data={[]}/>)

        expect(leader()).toBeNull()
        expect(screen.queryByRole("table")).toBeNull()
    })

    it("draws a country's flag from the atlas, never as an emoji", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("jp", 600), entry("fr", 500)]}/>)

        for (const place of [leader()!, rows()[0]]) {
            expect(place.querySelectorAll(".country-flag")).toHaveLength(1)
            expect(place.textContent).not.toMatch(/\p{RI}|\p{Extended_Pictographic}/u)
        }
    })

    it("does not chop a country name to fit its flag", () => {
        render(<Leaderboard tilesCount={100} data={[entry("gb-eng", 5)]}/>)
        expect(screen.getByText("England")).toBeDefined()
    })

    it("names itself, and labels its columns in words", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)

        expect(screen.getByRole("region", {name: "Leaderboard"})).toBeDefined()
        headers("#", "Country", "Tiles")
    })

    it("marks the player's own row", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 500), entry("jp", 250)]}
                            highlight={Countries.get("jp")!}/>)

        const marked = rows().filter(r => r.getAttribute("aria-current") === "true")
        expect(marked).toHaveLength(1)
        expect(within(marked[0]).getByText("Japan")).toBeDefined()
        expect(leader()!.getAttribute("aria-current")).toBeNull()
    })

    it("marks the frame when the player's country leads", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]} highlight={Countries.get("fr")!}/>)

        expect(leader()!.getAttribute("aria-current")).toBe("true")
        expect(rows().filter(r => r.getAttribute("aria-current") === "true")).toEqual([])
    })

    it("marks nothing when the player's country holds no tile", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 500), entry("de", 100)]}
                            highlight={Countries.get("jp")!}/>)

        expect(leader()!.getAttribute("aria-current")).toBeNull()
        expect(rows().filter(r => r.getAttribute("aria-current") === "true")).toEqual([])
    })

    it("owns no toggle of its own", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)
        expect(screen.queryAllByRole("button").filter(b => b.hasAttribute("aria-pressed"))).toEqual([])
    })

    it("says what a column means for a while when its head is tapped", () => {
        vi.useFakeTimers()
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)
        const head = screen.getByRole("button", {name: "Tiles"})

        fireEvent.pointerDown(head, {pointerType: "touch"})
        fireEvent.click(head)
        expect(screen.getByRole("status").textContent).toBe(t(FIGURES.tiles))

        act(() => vi.advanceTimersByTime(TAP_HINT_MS - 1))
        expect(screen.getByRole("status")).toBeDefined()
        act(() => vi.advanceTimersByTime(1))
        expect(screen.queryByRole("status")).toBeNull()
    })

    it("puts the bubble away at the next tap, on its own head or anywhere else", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)
        const head = screen.getByRole("button", {name: "Tiles"})
        const tap = (target: Element) => {
            fireEvent.pointerDown(target, {pointerType: "touch"})
            fireEvent.click(target)
        }

        tap(head)
        expect(screen.getByRole("status")).toBeDefined()
        tap(head)
        expect(screen.queryByRole("status")).toBeNull()

        tap(head)
        fireEvent.pointerDown(document.body, {pointerType: "touch"})
        expect(screen.queryByRole("status")).toBeNull()
    })

    it("keeps saying it while the mouse stays, even after a click", () => {
        vi.useFakeTimers()
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)
        const head = screen.getByRole("button", {name: "Tiles"})

        fireEvent.pointerEnter(head, {pointerType: "mouse"})
        fireEvent.pointerDown(head, {pointerType: "mouse"})
        fireEvent.click(head)
        act(() => vi.advanceTimersByTime(10 * TAP_HINT_MS))

        expect(screen.getByRole("status").textContent).toBe(t(FIGURES.tiles))
    })

    it("says what a column means while a mouse is over its head, and stops when it leaves", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)
        const head = screen.getByRole("button", {name: "Tiles"})

        fireEvent.pointerEnter(head, {pointerType: "mouse"})
        expect(screen.getByRole("status").textContent).toBe(t(FIGURES.tiles))

        fireEvent.pointerLeave(head, {pointerType: "mouse"})
        expect(screen.queryByRole("status")).toBeNull()
    })

    it("holds what it is given beside the first country", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 250)]} anthem={<p>the anthem</p>}/>)
        expect(within(leader()!).getByText("the anthem")).toBeDefined()
    })
})

describe("Leaderboard tile deltas", () => {
    const arrows = () => [leader()!, ...rows()].map(r => r.querySelector(".leaderboard-count .leaderboard-delta"))

    it("puts an arrow on the count of each country that just moved", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 503), entry("jp", 250), entry("de", 100)]}
                            deltas={deltas({fr: 3, jp: -2})}/>)

        expect(arrows().map((arrow) => arrow !== null)).toEqual([true, true, false])
    })

    it("points a loss the other way from a gain", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 503), entry("jp", 250), entry("de", 100)]}
                            deltas={deltas({fr: 3, jp: 2, de: -2})}/>)

        const [won, wonToo, lost] = arrows().map((arrow) => arrow!.innerHTML)
        expect(wonToo).toBe(won)
        expect(lost).not.toBe(won)
    })

    it("puts no arrow up while the board is still", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 500), entry("jp", 250)]}/>)
        expect(arrows()).toEqual([null, null])
    })

    it("colours the count itself the way the arrow points", () => {
        render(<Leaderboard tilesCount={1000}
                            data={[entry("fr", 600), entry("jp", 503), entry("de", 248), entry("gb-eng", 10)]}
                            deltas={deltas({jp: 3, de: -2})}/>)

        expect(rows().map(r => r.querySelector(".leaderboard-table-tiles")!.className))
            .toEqual([
                expect.stringContaining("leaderboard-tiles-up"),
                expect.stringContaining("leaderboard-tiles-down"),
                expect.not.stringContaining("leaderboard-tiles-"),
            ])
    })

    it("keeps the arrow out of what a screen reader reads", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 503), entry("jp", 250)]} deltas={deltas({fr: 3, jp: 1})}/>)
        expect(arrows().map((arrow) => arrow!.getAttribute("aria-hidden"))).toEqual(["true", "true"])
    })

    it("writes nothing between the count, its share and the next column", () => {
        render(<Leaderboard tilesCount={1000} data={[entry("fr", 600), entry("jp", 503)]} deltas={deltas({jp: 3})}/>)
        expect(cells()).toEqual([["2", "Japan", "503 50%"]])
    })
})

describe("Leaderboard with the season's race", () => {
    const MAP = [entry("de", 500), entry("fr", 300), entry("es", 100)]
    const RACE: Race = {
        today: {
            roundId: "season-0/day-5",
            times: 1,
            startedAt: 0,
            endedAt: 0,
            standings: [
                {rank: 1, countryCode: "de", share: 0.5, points: 25},
                {rank: 2, countryCode: "fr", share: 0.3, points: 18},
            ],
            takes: [],
            players: [],
            before: [],
            after: [],
            start: [{countryCode: "de", tiles: 400}, {countryCode: "fr", tiles: 310}],
        },
        scores: [{rank: 1, countryCode: "fr", points: 43, roundsWon: 1}],
    }

    beforeEach(() => useRace.setState({race: RACE}))

    it("ranks by the season first: the points, with what each would score if the day ended now", () => {
        render(<Leaderboard tilesCount={1000} data={MAP}/>)

        expect(leader()!.getAttribute("aria-label")).toBe("First: France")
        expect(leader()!.querySelector(".leader-frame-tiles")!.textContent).toBe("300 tiles 30%")
        expect(within(leader()!).getByText("points")).toBeDefined()
        expect(within(leader()!).getByText(/^43/).textContent).toBe("43+18 today")
        headers("#", "Country", "Tiles", "Gained", "Points", "Today")
        expect(cells()).toEqual([
            ["2", "Germany", "500 50%", "+100", "0", "+25 today"],
            ["3", "Spain", "100 10%", "+100", "0", ""],
        ])
    })

    it("heads the round in progress as the battle's, at its times, while a battle's round runs", () => {
        const battle: Race = {...RACE, battle: {...RACE.today!, roundId: "entry-1", times: 3, standings: [
            {rank: 1, countryCode: "de", share: 0.5, points: 75},
            {rank: 2, countryCode: "fr", share: 0.3, points: 54},
        ]}}
        useRace.setState({race: battle})
        render(<Leaderboard tilesCount={1000} data={MAP}/>)

        headers("#", "Country", "Tiles", "Gained", "Points", "Battle ×3")
        expect(screen.queryByRole("columnheader", {name: "Today"})).toBeNull()
        expect(within(leader()!).getByText(/^43/).textContent).toBe("43+54 this battle")
        expect(within(leader()!).getByRole("button", {name: "-10 this battle"})).toBeDefined()
        expect(cells()[0]).toEqual(["2", "Germany", "500 50%", "+100", "0", "+75 this battle"])
    })

    it("shows what each country gained since the day began, its tiles now less those it held then", () => {
        render(<Leaderboard tilesCount={1000} data={MAP}/>)

        expect(within(leader()!).getByRole("button", {name: "-10 today"})).toBeDefined()
        expect(leader()!.querySelector(".leaderboard-gain-down")).not.toBeNull()
        expect(rows()[0].querySelector(".leaderboard-gain-up")!.textContent).toBe("+100")
    })

    it("keeps the share beside the tiles while no round runs, with no column of its own", () => {
        useRace.setState({race: {scores: RACE.scores}})
        render(<Leaderboard tilesCount={1000} data={MAP}/>)

        expect(leader()!.querySelector(".leader-frame-tiles")!.textContent).toBe("300 tiles 30%")
        headers("#", "Country", "Tiles", "Points", "Today")
        expect(cells()[0]).toEqual(["2", "Germany", "500 50%", "0", ""])
    })

    it("says what the points mean the first time it shows them, once", () => {
        useBoardGuide.setState({guided: false})
        render(<Leaderboard tilesCount={1000} data={MAP}/>)

        expect(screen.getByRole("status").textContent).toBe(t(FIGURES.points))
        expect(useBoardGuide.getState().guided).toBe(true)
        expect(localStorage.getItem("clickplanet-board-guide")).toBe("points")
    })

    it("lets the first word on the points stay long enough to read", () => {
        vi.useFakeTimers()
        useBoardGuide.setState({guided: false})
        render(<Leaderboard tilesCount={1000} data={MAP}/>)

        act(() => vi.advanceTimersByTime(GUIDE_MS - 1))
        expect(screen.getByRole("status")).toBeDefined()
        act(() => vi.advanceTimersByTime(1))
        expect(screen.queryByRole("status")).toBeNull()
    })

    it("says nothing on its own once the player was told", () => {
        render(<Leaderboard tilesCount={1000} data={MAP}/>)

        expect(screen.queryByRole("status")).toBeNull()
    })

    const sortable = () => ["#", "Country", "Tiles", "Gained", "Points", "Today", "Battle ×3"].flatMap((name) => {
        const head = screen.queryByRole("columnheader", {name})
        return head?.querySelector(".leaderboard-sort") ? [[name, head.getAttribute("aria-sort")]] : []
    })

    it("sorts by the head picked among the tiles, the gain, the points and today's, and says which", () => {
        const onOrder = vi.fn()
        render(<Leaderboard tilesCount={1000} data={MAP} order="points" onOrder={onOrder}/>)

        expect(sortable()).toEqual([["Tiles", null], ["Gained", null], ["Points", "descending"], ["Today", null]])

        fireEvent.click(screen.getByRole("button", {name: "Gained"}))
        expect(onOrder).toHaveBeenCalledWith("gain")
        fireEvent.click(screen.getByRole("button", {name: "Today"}))
        expect(onOrder).toHaveBeenCalledWith("today")
    })

    // Germany 1st on tiles + 1st on gained, Spain 3rd + 1st, France 2nd + 3rd.
    const DAY: Race = {...RACE, today: {...RACE.today!, standings: [
        {rank: 1, countryCode: "de", share: 0.5, points: 25},
        {rank: 2, countryCode: "es", share: 0.1, points: 18},
        {rank: 3, countryCode: "fr", share: 0.3, points: 15},
    ]}}

    it("ranks by today's places when ordered by today, with each country's rank under its tiles and its gain", () => {
        useRace.setState({race: DAY})
        render(<Leaderboard tilesCount={1000} data={MAP} order="today" onOrder={vi.fn()}/>)

        expect(sortable()).toContainEqual(["Today", "descending"])
        expect(leader()!.getAttribute("aria-label")).toBe("First: Germany")
        expect(leader()!.querySelector(".leader-frame-ranks")!.textContent).toBe("1st on tiles + 1st on gained")
        expect(cells()).toEqual([
            ["2", "Spain", "100 10%3rd on tiles", "+1001st on gained", "0", "+18 today"],
            ["3", "France", "300 30%2nd on tiles", "-103rd on gained", "43", "+15 today"],
        ])
    })

    it("shows no ranks in another order", () => {
        useRace.setState({race: DAY})
        render(<Leaderboard tilesCount={1000} data={MAP} order="gain"/>)

        expect(leader()!.querySelector(".leader-frame-ranks")).toBeNull()
        expect(screen.queryAllByText(/on tiles|on gained/)).toEqual([])
    })

    it("ranks by the battle's places when ordered by today while a battle runs, with no ranks: it places the gain alone", () => {
        useRace.setState({race: {...RACE, battle: {...DAY.today!, roundId: "entry-1", times: 3, standings: [
            {rank: 1, countryCode: "es", share: 0.1, points: 75},
            {rank: 2, countryCode: "de", share: 0.5, points: 54},
        ]}}})
        render(<Leaderboard tilesCount={1000} data={MAP} order="today" onOrder={vi.fn()}/>)

        expect(sortable()).toContainEqual(["Battle ×3", "descending"])
        expect(leader()!.getAttribute("aria-label")).toBe("First: Spain")
        expect(leader()!.querySelector(".leader-frame-ranks")).toBeNull()
        expect(cells()).toEqual([
            ["2", "Germany", "500 50%", "+100", "0", "+54 this battle"],
            ["3", "France", "300 30%", "-10", "43", ""],
        ])
    })

    it("offers no gain to sort by while no round runs", () => {
        useRace.setState({race: {scores: RACE.scores}})
        render(<Leaderboard tilesCount={1000} data={MAP} order="gain" onOrder={vi.fn()}/>)

        expect(sortable()).toEqual([["Tiles", null], ["Points", "descending"]])
    })

    it("offers no sort with no race, or with no one to tell", () => {
        const {rerender} = render(<Leaderboard tilesCount={1000} data={MAP}/>)
        expect(sortable()).toEqual([])

        useRace.setState({race: undefined})
        rerender(<Leaderboard tilesCount={1000} data={MAP} onOrder={vi.fn()}/>)
        expect(sortable()).toEqual([])
    })

    it("ranks by the tiles gained since the day began when ordered by the gain", () => {
        render(<Leaderboard tilesCount={1000} data={MAP} order="gain"/>)

        expect(leader()!.getAttribute("aria-label")).toBe("First: Germany")
        expect(cells()).toEqual([
            ["2", "Spain", "100 10%", "+100", "0", ""],
            ["3", "France", "300 30%", "-10", "43", "+18 today"],
        ])
    })

    it("ranks by the tiles held now when ordered by the tiles, the points still beside them", () => {
        render(<Leaderboard tilesCount={1000} data={MAP} order="tiles"/>)

        expect(leader()!.getAttribute("aria-label")).toBe("First: Germany")
        expect(cells()).toEqual([
            ["2", "France", "300 30%", "-10", "43", "+18 today"],
            ["3", "Spain", "100 10%", "+100", "0", ""],
        ])
    })
})
