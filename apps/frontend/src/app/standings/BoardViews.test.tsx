// @vitest-environment jsdom
import {useState} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AccountBackend, Provider} from "../../backends/account.ts"
import {NameColor, PlayerBackend, PlayerLine, PlayerTitle} from "../../backends/player.ts"
import {MySeason, Standing, StandingsBackend} from "../../backends/standings.ts"
import {Countries, Country} from "../../domain/countries.ts"
import {hueOf} from "../../domain/authorColor.ts"
import {AccountStore} from "../account/accountStore.ts"
import SignInPitchModal from "../account/SignInPitchModal.tsx"
import {AcceptedClicks, acceptedClicks} from "../viewer/acceptedClicks.ts"
import BoardViews, {BoardView} from "./BoardViews.tsx"
import {Caller} from "./useMySeason.ts"

const france = Countries.get("fr")!
const germany = Countries.get("de")!

const standing = (rank: number, name: string, countryCode: string, tiles: number, color = NameColor.UNSPECIFIED): Standing =>
    ({rank, name, color, countryCode, tiles})

const WARLORD: PlayerTitle = {id: "warlord", name: "Warlord", rank: {trackId: "conquest", trackName: "Conquest", number: 3, count: 5}}
const DEVOTED: PlayerTitle = {id: "devoted", name: "Devoted", rank: {trackId: "devotion", trackName: "Devotion", number: 2, count: 3}}

const WORLD = [
    {...standing(1, "Ana", "fr", 1840, NameColor.PINK), wornTitle: WARLORD},
    standing(2, "kiran_07", "in", 1512),
    standing(2, "Mateus", "br", 1512, NameColor.TEAL),
]
const FRENCH = [standing(1, "Ana", "fr", 1840, NameColor.PINK), standing(2, "Bastien", "fr", 402)]
const FULL = [
    ...WORLD,
    standing(4, "zoe_nz", "nz", 1207),
    standing(5, "Jean Moulin", "fr", 990),
    standing(6, "Ольга", "ru", 864),
    standing(7, "Kofi", "gh", 731),
    standing(8, "Lucía", "es", 655),
    standing(9, "Hana", "jp", 590),
    standing(10, "Pierre_L", "fr", 13),
]
const FULL_FRENCH = Array.from({length: 10}, (_, i) => standing(i + 1, `fr_${i + 1}`, "fr", 500 - i * 40))

const GUEST: Caller = {linked: false}
const ZED: Caller = {linked: true, username: "Zed", color: NameColor.GREEN}

type Mine = (countryCode: string) => MySeason

const season = (main: MySeason, countries: Record<string, MySeason> = {}): Mine => (countryCode) =>
    countryCode === "" ? main : countries[countryCode] ?? {countryCode, tiles: 0}

function backendOf(mine?: Mine, world = WORLD, french = FRENCH) {
    return {
        listenForStandings: vi.fn((countryCode: string, onStandings: (standings: Standing[]) => void) => {
            void Promise.resolve().then(() => onStandings(countryCode === "" ? world : countryCode === "fr" ? french : []))
            return () => {}
        }),
        mySeason: vi.fn(async (countryCode: string) => mine?.(countryCode)),
    } satisfies StandingsBackend
}

type HarnessProps = {
    backend: StandingsBackend
    caller: Caller
    country?: Country
    view?: BoardView
    clicks?: AcceptedClicks
    onSignIn?: () => void
    onOpenPlayer?: (player: PlayerLine) => void
}

function Harness({backend, caller, country = france, view: first = "countries", clicks, onSignIn, onOpenPlayer}: HarnessProps) {
    const [view, setView] = useState<BoardView>(first)
    const [own] = useState(acceptedClicks)
    return <BoardViews backend={backend}
                       caller={caller}
                       listenForClicks={(clicks ?? own).listenForClicks}
                       onSignIn={onSignIn}
                       onOpenPlayer={onOpenPlayer}
                       view={view}
                       onView={setView}
                       country={country}
                       countries={<p>the countries</p>}/>
}

async function shown(props: HarnessProps) {
    const view = render(<Harness {...props}/>)
    await act(async () => {})
    return {...view, user: userEvent.setup()}
}

type User = ReturnType<typeof userEvent.setup>

const heading = (name: string) => screen.getByRole("button", {name: `Leaderboard: ${name}`})
const options = () => screen.getAllByRole("option")
async function pick(user: User, from: string, to: string) {
    await user.click(heading(from))
    await user.click(screen.getByRole("option", {name: to}))
}
const rows = () => screen.getAllByRole("row").slice(1)
const cells = () => rows().map((row) => within(row).getAllByRole("cell").map((cell) => cell.textContent))
const yours = () => screen.queryByRole("region", {name: "Your season"})
const stats = () => [...yours()!.querySelectorAll(".stat-tile")].map((tile) =>
    [tile.querySelector("dt")!.textContent, tile.querySelector("dd")!.textContent])

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("BoardViews", () => {
    it("starts on the countries, and offers the players and the players of the country played for", async () => {
        const {user} = await shown({backend: backendOf(), caller: GUEST})

        expect(screen.getByText("the countries")).toBeDefined()
        await user.click(heading("Countries"))
        expect(options().map((option) => option.textContent)).toEqual(["Countries", "Players", "France"])
        expect(screen.getByRole("option", {name: "Countries"}).getAttribute("aria-selected")).toBe("true")
        expect(screen.queryByRole("table")).toBeNull()
    })

    it("lists the players of the whole map, then those of the country played for", async () => {
        const backend = backendOf()
        const {user} = await shown({backend, caller: GUEST})

        await pick(user, "Countries", "Players")
        expect(heading("Players")).toBeDefined()
        expect(screen.queryByRole("listbox")).toBeNull()
        expect(screen.queryByText("the countries")).toBeNull()
        expect(backend.listenForStandings).toHaveBeenLastCalledWith("", expect.any(Function))
        expect(screen.getByRole("table", {name: "Players"})).toBeDefined()
        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "kiran_07", "1512"], ["2", "Mateus", "1512"]])

        await pick(user, "Players", "France")
        expect(backend.listenForStandings).toHaveBeenLastCalledWith("fr", expect.any(Function))
        expect(screen.getByRole("table", {name: "Players, France"})).toBeDefined()
        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "Bastien", "402"]])
    })

    it("moves another player's row as soon as the stream sends a new board", async () => {
        let send: (standings: Standing[]) => void = () => {}
        const backend = {
            listenForStandings: vi.fn((_: string, onStandings: (standings: Standing[]) => void) => {
                send = onStandings
                onStandings(WORLD)
                return () => {}
            }),
            mySeason: vi.fn(async () => undefined),
        } satisfies StandingsBackend
        await shown({backend, caller: GUEST, view: "players"})

        act(() => send([
            standing(1, "Mateus", "br", 1900, NameColor.TEAL),
            standing(2, "Ana", "fr", 1840, NameColor.PINK),
            standing(3, "kiran_07", "in", 1512),
        ]))

        expect(cells()).toEqual([["1", "Mateus", "1900"], ["2", "Ana", "1840"], ["3", "kiran_07", "1512"]])
    })

    it("follows the country played for", async () => {
        const backend = backendOf()
        const view = await shown({backend, caller: GUEST, view: "country"})

        view.rerender(<Harness backend={backend} caller={GUEST} country={germany} view="country"/>)
        await act(async () => {})

        expect(heading("Germany")).toBeDefined()
        expect(backend.listenForStandings).toHaveBeenLastCalledWith("de", expect.any(Function))
        expect(screen.getByText("Nobody yet.")).toBeDefined()
    })

    it("draws each player's main flag and its name in its color, grey with none", async () => {
        await shown({backend: backendOf(), caller: GUEST, view: "players"})

        const [ana, kiran] = rows()
        expect(within(ana).getByRole("img", {name: "France"}).querySelectorAll(".country-flag")).toHaveLength(1)
        expect(within(kiran).getByRole("img", {name: "India"})).toBeDefined()
        expect(within(ana).getByText("Ana").style.getPropertyValue("--author-hue")).toBe(String(hueOf(NameColor.PINK)))
        expect(within(kiran).getByText("kiran_07").style.getPropertyValue("--author-chroma")).toBe("0")
    })

    it("draws the title each player wears after its name, and none for a player who wears none", async () => {
        await shown({backend: backendOf(), caller: GUEST, view: "players"})

        const [ana, kiran] = rows()
        const medal = within(ana).getByRole("img", {name: "Warlord"})
        expect(medal.querySelector("svg")!.getAttribute("width")).toBe("18")
        expect(within(ana).getByText("Ana").nextElementSibling).toBe(medal)
        expect(kiran.querySelector(".title-badge")).toBeNull()
    })

    it("gives the first three their coins and shares a rank between ties", async () => {
        await shown({backend: backendOf(), caller: GUEST, view: "players"})

        expect(rows().map((row) => row.querySelector(".coin")?.className)).toEqual(["coin coin-1", "coin coin-2", "coin coin-2"])
    })

    it("marks the caller's own row when it is in the top 10, and adds no line under it", async () => {
        const mine = season({countryCode: "br", tiles: 1512, rank: 2})
        await shown({backend: backendOf(mine), caller: {linked: true, username: "Mateus", color: NameColor.TEAL}, view: "players"})

        const marked = rows().filter((row) => row.getAttribute("aria-current") === "true")
        expect(marked).toHaveLength(1)
        expect(within(marked[0]).getByText("Mateus")).toBeDefined()
        expect(marked[0].querySelector(".coin-you")!.textContent).toBe("2")
        expect(rows()).toHaveLength(3)
    })

    it("marks nothing for a guest", async () => {
        await shown({backend: backendOf(season({countryCode: "fr", tiles: 3})), caller: GUEST, view: "players"})
        expect(rows().filter((row) => row.getAttribute("aria-current") === "true")).toEqual([])
    })

    it("puts the caller's own line under the top 10, at its rank on the whole map", async () => {
        const mine = season({countryCode: "fr", tiles: 12, rank: 42}, {fr: {countryCode: "fr", tiles: 12, rank: 7}})
        await shown({backend: backendOf(mine, FULL), caller: ZED, view: "players"})

        expect(cells()).toHaveLength(11)
        expect(cells().slice(-2)).toEqual([["10", "Pierre_L", "13"], ["42", "Zed", "12"]])
        const own = rows()[10]
        expect(own.getAttribute("aria-current")).toBe("true")
        expect(within(own).getByRole("img", {name: "France"})).toBeDefined()
        expect(within(own).getByText("Zed").style.getPropertyValue("--author-hue")).toBe(String(hueOf(NameColor.GREEN)))
    })

    it("puts the title the caller wears on its own line", async () => {
        const mine = season({countryCode: "fr", tiles: 12, rank: 42, wornTitle: DEVOTED})
        await shown({backend: backendOf(mine), caller: ZED, view: "players"})

        expect(within(rows()[3]).getByRole("img", {name: "Devoted"})).toBeDefined()
    })

    it("puts the caller's line under its country's top 10 at its rank there", async () => {
        const mine = season({countryCode: "fr", tiles: 12, rank: 42}, {fr: {countryCode: "fr", tiles: 12, rank: 17}})
        await shown({backend: backendOf(mine, WORLD, FULL_FRENCH), caller: ZED, view: "country"})

        expect(cells()).toHaveLength(11)
        expect(cells().at(-1)).toEqual(["17", "Zed", "12"])
    })

    it("lists a caller missing from a country list shorter than 10 in it", async () => {
        const mine = season({countryCode: "fr", tiles: 12, rank: 42}, {fr: {countryCode: "fr", tiles: 12, rank: 3}})
        await shown({backend: backendOf(mine), caller: ZED, view: "country"})

        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "Bastien", "402"], ["3", "Zed", "12"]])
    })

    it("lists the caller on the board of a country it took tiles for, though another flag is its main one", async () => {
        const mine = season({countryCode: "bg", tiles: 74, rank: 1}, {fr: {countryCode: "fr", tiles: 3, rank: 3}})
        await shown({backend: backendOf(mine), caller: ZED, view: "country"})

        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "Bastien", "402"], ["3", "Zed", "3"]])
        expect(within(rows()[2]).getByRole("img", {name: "France"})).toBeDefined()
        expect(screen.queryByRole("img", {name: "Bulgaria"})).toBeNull()
    })

    it("asks for the caller's line on the board shown, and again when another is picked", async () => {
        const backend = backendOf(season({countryCode: "fr", tiles: 12, rank: 42}))
        const {user} = await shown({backend, caller: ZED, view: "players"})
        expect(backend.mySeason).toHaveBeenLastCalledWith("")

        await pick(user, "Players", "France")

        expect(backend.mySeason).toHaveBeenLastCalledWith("fr")
    })

    it("counts each tile the caller takes into its row and its season at once", async () => {
        const clicks = acceptedClicks()
        const mine = season({countryCode: "br", tiles: 1512, rank: 2})
        await shown({backend: backendOf(mine), caller: {linked: true, username: "Mateus", color: NameColor.TEAL}, view: "players", clicks})

        await act(() => clicks.record({country: "br", took: true}))

        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "Mateus", "1513"], ["3", "kiran_07", "1512"]])
        expect(stats()).toEqual([["Tiles", "1513"], ["Players", "#2"]])
    })

    it("counts each tile the caller takes for the country shown into its row there, and no other", async () => {
        const clicks = acceptedClicks()
        const mine = season({countryCode: "bg", tiles: 74, rank: 1}, {fr: {countryCode: "fr", tiles: 400, rank: 3}})
        await shown({backend: backendOf(mine), caller: ZED, view: "country", clicks})

        await act(() => clicks.record({country: "fr", took: true}))
        await act(() => clicks.record({country: "fr", took: true}))
        await act(() => clicks.record({country: "fr", took: true}))
        await act(() => clicks.record({country: "bg", took: true}))

        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "Zed", "403"], ["3", "Bastien", "402"]])
        expect(stats()).toEqual([["Tiles", "403"], ["France", "#2"]])
    })

    it("lets the caller into the top 10 once it passes the last of it", async () => {
        const clicks = acceptedClicks()
        const mine = season({countryCode: "fr", tiles: 12, rank: 42})
        await shown({backend: backendOf(mine, FULL), caller: ZED, view: "players", clicks})

        await act(() => clicks.record({country: "fr", took: true}))
        await act(() => clicks.record({country: "fr", took: true}))

        expect(cells()).toHaveLength(10)
        expect(cells().at(-1)).toEqual(["10", "Zed", "14"])
        expect(stats()).toEqual([["Tiles", "14"], ["Players", "#10"]])
    })

    it("opens a player's card from its name, with the title it wears", async () => {
        const onOpenPlayer = vi.fn()
        const {user} = await shown({backend: backendOf(), caller: GUEST, view: "players", onOpenPlayer})

        await user.click(within(rows()[0]).getByRole("button", {name: "Ana"}))

        expect(onOpenPlayer).toHaveBeenCalledWith({
            name: "Ana", countryCode: "fr", guest: false, admin: false, color: NameColor.PINK, streak: 0, wornTitle: WARLORD,
        })
    })

    it("draws the names as plain text with no card to open", async () => {
        await shown({backend: backendOf(), caller: GUEST, view: "players"})
        expect(within(rows()[0]).queryByRole("button")).toBeNull()
    })

    it("leaves the caller's line out of a country it took nothing for", async () => {
        const mine = season({countryCode: "fr", tiles: 12, rank: 42}, {fr: {countryCode: "fr", tiles: 12, rank: 7}})
        await shown({backend: backendOf(mine), caller: ZED, country: germany, view: "country"})

        expect(screen.getByText("Nobody yet.")).toBeDefined()
        expect(screen.queryByRole("table")).toBeNull()
    })

    it("leaves the caller's line out while it has no rank", async () => {
        await shown({backend: backendOf(season({tiles: 0})), caller: ZED, view: "players"})
        expect(cells()).toHaveLength(3)
    })
})

describe("Your season", () => {
    it("is not on the countries' board", async () => {
        await shown({backend: backendOf(season({countryCode: "fr", tiles: 12, rank: 42})), caller: ZED})
        expect(yours()).toBeNull()
    })

    it("says a named player's season tiles and its rank among all players", async () => {
        const mine = season({countryCode: "fr", tiles: 12, rank: 42}, {de: {countryCode: "de", tiles: 2, rank: 7}})
        await shown({backend: backendOf(mine, FULL), caller: ZED, country: germany, view: "players"})

        expect(stats()).toEqual([["Tiles", "12"], ["Players", "#42"]])
        expect(within(yours()!).queryByRole("button")).toBeNull()
    })

    it("says only the tiles a named player took for the country shown, and its rank there", async () => {
        const mine = season({countryCode: "bg", tiles: 74, rank: 1}, {fr: {countryCode: "fr", tiles: 12, rank: 7}})
        await shown({backend: backendOf(mine, WORLD, FULL_FRENCH), caller: ZED, view: "country"})

        expect(stats()).toEqual([["Tiles", "12"], ["France", "#7"]])
        expect(yours()!.textContent).not.toContain("Bulgaria")
    })

    it("shows a dash for a rank a named player does not hold yet", async () => {
        await shown({backend: backendOf(season({tiles: 0})), caller: ZED, view: "players"})
        expect(stats()).toEqual([["Tiles", "0"], ["Players", "—"]])

        cleanup()
        await shown({backend: backendOf(season({countryCode: "bg", tiles: 74, rank: 1})), caller: ZED, view: "country"})
        expect(stats()).toEqual([["Tiles", "0"], ["France", "—"]])
    })

    it("gives a guest its tiles and a Sign in button, and no rank", async () => {
        const onSignIn = vi.fn()
        const {user} = await shown({backend: backendOf(season({countryCode: "fr", tiles: 9})), caller: GUEST, view: "players", onSignIn})

        expect(stats()).toEqual([["Tiles", "9"]])
        await user.click(within(yours()!).getByRole("button", {name: "Sign in"}))
        expect(onSignIn).toHaveBeenCalledTimes(1)
    })

    it("still offers a guest whose tiles are not known yet to sign in", async () => {
        await shown({backend: backendOf(undefined), caller: GUEST, view: "players", onSignIn: vi.fn()})

        expect(stats()).toEqual([["Tiles", "—"]])
        expect(within(yours()!).getByRole("button", {name: "Sign in"})).toBeDefined()
    })

    it("offers no sign-in where none is offered, and says nothing while the season is unknown", async () => {
        await shown({backend: backendOf(undefined), caller: {linked: true}, view: "players"})
        expect(yours()).toBeNull()
    })

    it("opens the sign-in pitch from a guest's Sign in button", async () => {
        const store = await guestStore()
        const state = store.state()
        if (state.kind !== "ready") throw new Error(`the store is ${state.kind}`)

        function WithPitch() {
            const [open, setOpen] = useState(false)
            return <>
                <Harness backend={backendOf(season({countryCode: "fr", tiles: 9}))} caller={GUEST} view="players" onSignIn={() => setOpen(true)}/>
                {open && state.kind === "ready" && <SignInPitchModal state={state} store={store} multiplier={2} onClose={() => setOpen(false)}/>}
            </>
        }

        render(<WithPitch/>)
        await act(async () => {})
        await userEvent.setup().click(within(yours()!).getByRole("button", {name: "Sign in"}))

        expect(screen.getByRole("dialog", {name: "Sign in and stand out"})).toBeDefined()
    })
})

async function guestStore() {
    const backend = {
        signInOptions: vi.fn(async (): Promise<Provider[]> => ["google"]),
        me: vi.fn(async () => ({linked: []})),
        startSignIn: vi.fn(async () => "https://google.example/authorize"),
        completeSignIn: vi.fn(async () => undefined),
        startEmailSignIn: vi.fn(async () => undefined),
        completeEmailSignIn: vi.fn(async () => undefined),
        signOut: vi.fn(async () => undefined),
        signOutEverywhere: vi.fn(async () => undefined),
        deleteAccount: vi.fn(async () => undefined),
    } satisfies AccountBackend
    const player = {
        profile: vi.fn(async () => ({accountId: "account-1", name: "", color: NameColor.UNSPECIFIED})),
        setName: vi.fn(async (name: string) => ({accountId: "account-1", name})),
        setColor: vi.fn(async (color: NameColor) => color),
        titles: vi.fn(async () => ({wearable: [], tracks: []})),
        fronts: vi.fn(async () => ({playsFor: [], playsAgainst: []})),
        wearTitle: vi.fn(async () => undefined),
    } satisfies PlayerBackend
    const store = new AccountStore(backend, player, {token: vi.fn(), held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn()}, {navigate: vi.fn(), remember: vi.fn()})
    await store.load()
    return store
}
