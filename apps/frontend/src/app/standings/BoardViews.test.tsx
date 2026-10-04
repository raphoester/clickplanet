// @vitest-environment jsdom
import {useState} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {AccountBackend, Provider} from "../../backends/account.ts"
import {NameColor, PlayerBackend} from "../../backends/player.ts"
import {MySeason, Standing, StandingsBackend} from "../../backends/standings.ts"
import {Countries, Country} from "../../domain/countries.ts"
import {hueOf} from "../../domain/authorColor.ts"
import {AccountStore} from "../account/accountStore.ts"
import SignInPitchModal from "../account/SignInPitchModal.tsx"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import BoardViews, {BoardView} from "./BoardViews.tsx"
import {Caller} from "./useMySeason.ts"

const france = Countries.get("fr")!
const germany = Countries.get("de")!

const standing = (rank: number, name: string, countryCode: string, tiles: number, color = NameColor.UNSPECIFIED): Standing =>
    ({rank, name, color, countryCode, tiles})

const WORLD = [
    standing(1, "Ana", "fr", 1840, NameColor.PINK),
    standing(2, "kiran_07", "in", 1512),
    standing(2, "Mateus", "br", 1512, NameColor.TEAL),
]
const FRENCH = [standing(1, "Ana", "fr", 1840, NameColor.PINK), standing(2, "Bastien", "fr", 402)]

const NO_CLICKS: ListenForClicks = () => () => {}

const GUEST: Caller = {linked: false}
const ZED: Caller = {linked: true, username: "Zed", color: NameColor.GREEN}

function backendOf(mine?: MySeason) {
    return {
        standings: vi.fn(async (countryCode: string) => countryCode === "" ? WORLD : countryCode === "fr" ? FRENCH : []),
        mySeason: vi.fn(async () => mine),
    } satisfies StandingsBackend
}

type HarnessProps = {
    backend: StandingsBackend
    caller: Caller
    country?: Country
    view?: BoardView
    onSignIn?: () => void
}

function Harness({backend, caller, country = france, view: first = "countries", onSignIn}: HarnessProps) {
    const [view, setView] = useState<BoardView>(first)
    return <BoardViews backend={backend}
                       caller={caller}
                       listenForClicks={NO_CLICKS}
                       onSignIn={onSignIn}
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

const tab = (name: string) => screen.getByRole("tab", {name})
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
        await shown({backend: backendOf(), caller: GUEST})

        expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Countries", "Players", "France"])
        expect(tab("Countries").getAttribute("aria-selected")).toBe("true")
        expect(screen.getByText("the countries")).toBeDefined()
        expect(screen.queryByRole("table")).toBeNull()
    })

    it("lists the players of the whole map, then those of the country played for", async () => {
        const backend = backendOf()
        const {user} = await shown({backend, caller: GUEST})

        await user.click(tab("Players"))
        expect(tab("Players").getAttribute("aria-selected")).toBe("true")
        expect(screen.queryByText("the countries")).toBeNull()
        expect(backend.standings).toHaveBeenLastCalledWith("")
        expect(screen.getByRole("table", {name: "Players"})).toBeDefined()
        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "kiran_07", "1512"], ["2", "Mateus", "1512"]])

        await user.click(tab("France"))
        expect(backend.standings).toHaveBeenLastCalledWith("fr")
        expect(screen.getByRole("table", {name: "Players, France"})).toBeDefined()
        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "Bastien", "402"]])
    })

    it("follows the country played for", async () => {
        const backend = backendOf()
        const view = await shown({backend, caller: GUEST, view: "country"})

        view.rerender(<Harness backend={backend} caller={GUEST} country={germany} view="country"/>)
        await act(async () => {})

        expect(tab("Germany").getAttribute("aria-selected")).toBe("true")
        expect(backend.standings).toHaveBeenLastCalledWith("de")
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

    it("gives the first three their coins and shares a rank between ties", async () => {
        await shown({backend: backendOf(), caller: GUEST, view: "players"})

        expect(rows().map((row) => row.querySelector(".coin")?.className)).toEqual(["coin coin-1", "coin coin-2", "coin coin-2"])
    })

    it("marks the caller's own row when it is in the top 10, and adds no line under it", async () => {
        const mine = {countryCode: "br", tiles: 1512, globalRank: 2, countryRank: 1}
        await shown({backend: backendOf(mine), caller: {linked: true, username: "Mateus", color: NameColor.TEAL}, view: "players"})

        const marked = rows().filter((row) => row.getAttribute("aria-current") === "true")
        expect(marked).toHaveLength(1)
        expect(within(marked[0]).getByText("Mateus")).toBeDefined()
        expect(marked[0].querySelector(".coin-you")!.textContent).toBe("2")
        expect(rows()).toHaveLength(3)
    })

    it("marks nothing for a guest", async () => {
        await shown({backend: backendOf({countryCode: "fr", tiles: 3}), caller: GUEST, view: "players"})
        expect(rows().filter((row) => row.getAttribute("aria-current") === "true")).toEqual([])
    })

    it("puts the caller's own line under the top 10, at its rank on the whole map", async () => {
        const mine = {countryCode: "fr", tiles: 12, globalRank: 42, countryRank: 7}
        await shown({backend: backendOf(mine), caller: ZED, view: "players"})

        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "kiran_07", "1512"], ["2", "Mateus", "1512"], ["42", "Zed", "12"]])
        const own = rows()[3]
        expect(own.getAttribute("aria-current")).toBe("true")
        expect(within(own).getByRole("img", {name: "France"})).toBeDefined()
        expect(within(own).getByText("Zed").style.getPropertyValue("--author-hue")).toBe(String(hueOf(NameColor.GREEN)))
    })

    it("puts the caller's line under its country's top 10 at its rank there", async () => {
        const mine = {countryCode: "fr", tiles: 12, globalRank: 42, countryRank: 7}
        await shown({backend: backendOf(mine), caller: ZED, view: "country"})

        expect(cells()).toEqual([["1", "Ana", "1840"], ["2", "Bastien", "402"], ["7", "Zed", "12"]])
    })

    it("leaves the caller's line out of a country that is not its main flag", async () => {
        const mine = {countryCode: "fr", tiles: 12, globalRank: 42, countryRank: 7}
        await shown({backend: backendOf(mine), caller: ZED, country: germany, view: "country"})

        expect(screen.getByText("Nobody yet.")).toBeDefined()
        expect(screen.queryByRole("table")).toBeNull()
    })

    it("leaves the caller's line out while it has no rank", async () => {
        await shown({backend: backendOf({tiles: 0}), caller: ZED, view: "players"})
        expect(cells()).toHaveLength(3)
    })
})

describe("Your season", () => {
    it("is not on the countries' board", async () => {
        await shown({backend: backendOf({countryCode: "fr", tiles: 12, globalRank: 42, countryRank: 7}), caller: ZED})
        expect(yours()).toBeNull()
    })

    it("says a named player's tiles and its ranks on the map and in its main flag", async () => {
        await shown({backend: backendOf({countryCode: "fr", tiles: 12, globalRank: 42, countryRank: 7}), caller: ZED, country: germany, view: "players"})

        expect(stats()).toEqual([["Tiles", "12"], ["Players", "#42"], ["France", "#7"]])
        expect(within(yours()!).queryByRole("button")).toBeNull()
    })

    it("shows a dash for a rank a named player does not hold yet", async () => {
        await shown({backend: backendOf({tiles: 0}), caller: ZED, view: "players"})
        expect(stats()).toEqual([["Tiles", "0"], ["Players", "—"]])
    })

    it("gives a guest its tiles and a Sign in button, and no rank", async () => {
        const onSignIn = vi.fn()
        const {user} = await shown({backend: backendOf({countryCode: "fr", tiles: 9}), caller: GUEST, view: "players", onSignIn})

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
                <Harness backend={backendOf({countryCode: "fr", tiles: 9})} caller={GUEST} view="players" onSignIn={() => setOpen(true)}/>
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
        wearTitle: vi.fn(async () => undefined),
    } satisfies PlayerBackend
    const store = new AccountStore(backend, player, {token: vi.fn(), held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn()}, {navigate: vi.fn(), remember: vi.fn()})
    await store.load()
    return store
}
