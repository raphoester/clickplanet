// @vitest-environment jsdom
import {ReactNode} from "react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import Menu, {BoardSheet, YouPlace} from "./Menu.tsx"
import {Countries} from "../domain/countries.ts"
import type {LeaderboardEntry} from "../domain/leaderboard.ts"
import type {CountryOrder} from "../domain/race.ts"
import {DEFAULT_SOUND_SETTINGS} from "../domain/soundSettings.ts"
import {DEFAULT_DISPLAY_SETTINGS, DisplaySettings} from "../domain/displaySettings.ts"
import {useDisplaySettings} from "./settings/displaySettingsStore.ts"
import {useSoundSettings} from "./sound/soundSettingsStore.ts"
import {useBoardGuide} from "./boardGuideStore.ts"
import {useScreen} from "./hud/screenStore.ts"
import {useRace} from "./standings/raceStore.ts"
import {AccountBackend, Me, Provider} from "../backends/account.ts"
import {AccountStore} from "./account/accountStore.ts"
import {NameColor} from "../backends/player.ts"
import {PlayerBackend, PlayerError, PlayerInfoBackend} from "../backends/player.ts"
import {Fronts, PlayerTitle, TitleDashboard, ProgressionBackend} from "../backends/progression.ts"
import {AcceptedClicks, acceptedClicks} from "./clicks/acceptedClicks.ts"
import {SETTLE_MS} from "./clicks/useAcceptedClicks.ts"
import {Race} from "../backends/standings.ts"
import {BUILD} from "./support/pageContext.ts"
import {NO_ATTACHMENTS} from "../backends/support.ts"
import {AllegianceLockedError} from "../backends/backend.ts"
import {FakeAccountBackend} from "../backends/fakeAccountBackend.ts"
import {FakeClipsBackend} from "../backends/fakeClipsBackend.ts"
import {FakeSupportBackend} from "../backends/fakeSupportBackend.ts"
import {NoSession} from "../backends/session.ts"

const france = Countries.get("fr")!
const RACE: Race = {
    today: {
        roundId: "season-0/day-5", times: 1, startedAt: 0, endedAt: 0,
        standings: [{rank: 1, countryCode: "fr", share: 0.5, points: 25}], takes: [], players: [], before: [], after: [], start: [],
    },
    scores: [{rank: 1, countryCode: "fr", points: 43, roundsWon: 1}],
}
const entry = (code: string, tiles: number) => ({country: Countries.get(code)!, tiles})
const NO_PLAYER: PlayerInfoBackend = {playerInfo: async () => undefined}

function backends() {
    const account = new FakeAccountBackend()
    return {
        account: new AccountStore(account, account, account, new NoSession(), {navigate: vi.fn(), remember: vi.fn(), claimFlag: vi.fn()}),
        playerInfo: NO_PLAYER,
        clips: new FakeClipsBackend(),
        support: new FakeSupportBackend(),
    }
}

beforeEach(() => useBoardGuide.setState({guided: true}))

afterEach(() => {
    cleanup()
    localStorage.clear()
    useDisplaySettings.setState(DEFAULT_DISPLAY_SETTINGS)
    useSoundSettings.setState({settings: DEFAULT_SOUND_SETTINGS})
    useScreen.setState(useScreen.getInitialState(), true)
    useRace.setState({race: undefined, heard: undefined})
    vi.useRealTimers()
})

function setup(leaderboard: LeaderboardEntry[] = [], country = france) {
    const setCountry = vi.fn()
    const view = render(
        <Menu {...backends()} country={country} setCountry={setCountry} leaderboard={leaderboard} tilesCount={1000}/>,
    )
    return {...view, setCountry, user: userEvent.setup()}
}

const board = () => screen.queryByRole("region", {name: "Leaderboard"})
const countryPanel = () => screen.queryByRole("listbox", {name: "Country"})
const aboutDialog = () => screen.queryByRole("dialog", {name: "About ClickPlanet"})
const button = (name: string | RegExp) => screen.getByRole("button", {name})
const tab = (name: string) => screen.getByRole("tab", {name})

describe("Menu", () => {
    it("folds the panel away on its hide button, and opens it again on a place", async () => {
        const {user} = setup([entry("fr", 500)])

        await user.click(button("Hide the panel"))

        expect(board()).toBeNull()
        expect(screen.getAllByRole("tab").every((t) => t.getAttribute("aria-selected") === "false")).toBe(true)
        expect(button("Change")).toBeDefined()
        expect(localStorage.getItem("clickplanet-menu-open")).toBe("false")

        await user.click(tab("Board"))

        expect(board()).not.toBeNull()
        expect(tab("Board").getAttribute("aria-selected")).toBe("true")
    })

    it("folds the panel away when its place is pressed again", async () => {
        const {user} = setup([entry("fr", 500)])

        await user.click(tab("Board"))

        expect(board()).toBeNull()
    })

    it("opens folded when the player left it folded", () => {
        localStorage.setItem("clickplanet-menu-open", "false")
        setup([entry("fr", 500)])

        expect(board()).toBeNull()
    })

    it("opens the panel on the country picker from the folded flag", async () => {
        localStorage.setItem("clickplanet-menu-open", "false")
        const {user} = setup()

        await user.click(button("Change"))

        expect(countryPanel()).not.toBeNull()
    })

    it("offers the Hall of Fame as an icon in the board's header", () => {
        render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                     hallOfFame={{ended: [], standings: {} as never, turn: {} as never}}/>)

        expect(button("Hall of Fame").className).toContain("icon-button")
        expect(screen.queryByText("Hall of Fame")).toBeNull()
    })

    it("leads back to the home page from the logo", () => {
        setup()

        expect(screen.getByRole("link", {name: "ClickPlanet home page"}).getAttribute("href")).toBe("/#home")
    })

    it("counts the missions not seen yet on their tab", () => {
        render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                     missions={{state: {kind: "loading"}}} missionsNews={3}/>)

        expect(tab("Missions, 3 new").textContent).toBe("Missions3")
    })

    it("shows on the Change button how long the flag is held, and in the picker until when", async () => {
        const user = userEvent.setup()
        const changeableAt = Date.now() + 30 * 60 * 1000
        render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                     allegiance={{countryId: "fr", changeableAt}}/>)

        const time = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"}).format(changeableAt)
        const change = button(`Change country, next change at ${time}`)
        expect(change.textContent).toBe("30 min")

        await user.click(change)

        expect(screen.getByRole("status").textContent).toBe(`You can change country once an hour. Next change at ${time}.`)
        expect(countryPanel()).toBeNull()
    })

    it("shows a padlock and no list while the rules lock the flag, whatever the wait says", async () => {
        const user = userEvent.setup()
        render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                     allegiance={{countryId: "fr", changeableAt: Date.now() - 1, locked: true}}/>)

        const change = button("Change country, locked for now")
        expect(change.textContent).toBe("Locked")

        await user.click(change)

        expect(screen.getByRole("status").textContent).toBe("You can't change country right now.")
        expect(countryPanel()).toBeNull()
    })

    it("offers every country once the wait is over, under the rule", async () => {
        const user = userEvent.setup()
        render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                     allegiance={{countryId: "fr", changeableAt: Date.now() - 1}}/>)

        await user.click(button("Change"))

        expect(screen.getByRole("status").textContent).toBe("You can change country once an hour.")
        expect(countryPanel()).not.toBeNull()
    })

    it("stays in the picker when the change is refused, and says so when it failed", async () => {
        const user = userEvent.setup()
        const setCountry = vi.fn()
            .mockRejectedValueOnce(new AllegianceLockedError({countryId: "fr", changeableAt: Date.now() + 60_000}))
            .mockRejectedValueOnce(new Error("the network fell over"))
        vi.spyOn(console, "error").mockImplementation(() => {})
        render(<Menu {...backends()} country={france} setCountry={setCountry} leaderboard={[]} tilesCount={1000}/>)

        await user.click(button("Change"))
        await user.click(screen.getByRole("option", {name: "Japan"}))
        expect(screen.queryByRole("alert")).toBeNull()
        expect(countryPanel()).not.toBeNull()

        await user.click(screen.getByRole("option", {name: "Japan"}))
        expect(screen.getByRole("alert").textContent).toBe("Could not change country. Try again.")
        expect(countryPanel()).not.toBeNull()
        vi.mocked(console.error).mockRestore()
    })

    it("starts on the leaderboard, with the picker and About closed", () => {
        setup([entry("fr", 500)])

        expect(tab("Board").getAttribute("aria-selected")).toBe("true")
        expect(board()).not.toBeNull()
        expect(countryPanel()).toBeNull()
        expect(aboutDialog()).toBeNull()
    })

    it("draws the country you play for with a flag, not an emoji", () => {
        const {container} = setup([entry("fr", 500)])

        const playing = container.querySelector(".menu-playing-name")!
        expect(playing.querySelectorAll(".country-flag")).toHaveLength(1)
        expect(playing.querySelector(".menu-playing-country")!.textContent).toBe("France")
    })

    it("says the country's rank beside it", () => {
        setup([entry("jp", 500), entry("fr", 250)])
        expect(screen.getByLabelText("Rank 2").textContent).toBe("#2")
    })

    it("shows a dash rather than a rank when the country holds no tile", () => {
        setup([entry("jp", 500)])
        expect(screen.getByLabelText("No rank yet").textContent).toBe("—")
    })

    it("ranks the country in the order picked, as the board under it does", () => {
        useRace.setState({race: RACE})
        const menu = (countryOrder: CountryOrder) => <Menu {...backends()} country={france} setCountry={vi.fn()} tilesCount={1000}
                                                           leaderboard={[entry("jp", 500), entry("fr", 250)]}
                                                           countryOrder={countryOrder}/>

        const {rerender} = render(menu("points"))
        expect(screen.getByLabelText("Rank 1").textContent).toBe("#1")
        expect(screen.getByRole("region", {name: "First: France"})).toBeDefined()

        rerender(menu("tiles"))
        expect(screen.getByLabelText("Rank 2").textContent).toBe("#2")
        expect(screen.getByRole("region", {name: "First: Japan"})).toBeDefined()
    })

    it("ranks a country with points and no ground in the season, and not by territory", () => {
        useRace.setState({race: RACE})
        const menu = (countryOrder: CountryOrder) => <Menu {...backends()} country={france} setCountry={vi.fn()} tilesCount={1000}
                                                           leaderboard={[entry("jp", 500)]}
                                                           countryOrder={countryOrder}/>

        const {rerender} = render(menu("points"))
        expect(screen.getByLabelText("Rank 1").textContent).toBe("#1")

        rerender(menu("tiles"))
        expect(screen.getByLabelText("No rank yet").textContent).toBe("—")
    })

    it("offers the board, Share and the settings, and the account only when there is one", () => {
        setup()
        expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Board", "Share", "Settings"])
    })

    it("swaps the board for the place picked, in the same panel", async () => {
        const {user} = setup([entry("fr", 500)])

        await user.click(tab("Settings"))

        expect(tab("Settings").getAttribute("aria-selected")).toBe("true")
        expect(board()).toBeNull()
        expect(button("About")).toBeDefined()
    })

    describe("the leader", () => {
        const withLeader = (data: LeaderboardEntry[], anthem?: ReactNode) => render(
            <Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={data} tilesCount={1000} anthem={anthem}/>)

        it("is drawn in a frame of its own, apart from the table", () => {
            withLeader([entry("jp", 500), entry("fr", 250)])

            const leader = screen.getByRole("region", {name: "First: Japan"})
            expect(within(leader).getByText("50.00")).toBeDefined()
            expect(screen.getAllByRole("row").slice(1).map((r) => within(r).getAllByRole("cell")[1].textContent))
                .toEqual(["France"])
        })

        it("says nothing of the toll, which the dock shows", () => {
            withLeader([entry("jp", 500)])
            expect(screen.queryByText(/slower/)).toBeNull()
        })

        it("holds the anthem player", () => {
            withLeader([entry("jp", 500)], <p>the anthem</p>)
            expect(within(screen.getByRole("region", {name: "First: Japan"})).getByText("the anthem")).toBeDefined()
        })

        it("is marked when it is the player's own country", () => {
            withLeader([entry("fr", 500)])
            expect(screen.getByRole("region", {name: "First: France"}).getAttribute("aria-current")).toBe("true")
        })
    })

    describe("the players' standings", () => {
        const standings = {
            backend: {
                listenForStandings: vi.fn(() => () => {}),

                mySeason: vi.fn(async () => undefined),
            },
            caller: {linked: false},
            listenForClicks: () => () => {},
            view: "countries" as const,
            onView: vi.fn(),
        }

        it("are offered beside the countries once wired, and the countries stay as they were", () => {
            render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[entry("fr", 500)]} tilesCount={1000} standings={standings}/>)

            expect(screen.getByRole("button", {name: "Leaderboard: Countries"})).toBeDefined()
            expect(screen.getByRole("region", {name: "First: France"})).toBeDefined()
        })

        it("are not offered when none are wired", () => {
            setup([entry("fr", 500)])
            expect(screen.queryByRole("button", {name: /^Leaderboard/})).toBeNull()
        })

        it("let the countries' table pick its order from its heads", () => {
            useRace.setState({race: RACE})
            const onCountryOrder = vi.fn()
            render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[entry("jp", 500), entry("fr", 250)]}
                         tilesCount={1000} standings={standings} onCountryOrder={onCountryOrder}/>)

            fireEvent.click(button("Tiles"))
            expect(onCountryOrder).toHaveBeenCalledWith("tiles")
        })
    })

    describe("the board's sheet, on a phone", () => {
        const standings = {
            backend: {
                listenForStandings: vi.fn(() => () => {}),

                mySeason: vi.fn(async () => undefined),
            },
            caller: {linked: false},
            listenForClicks: () => () => {},
            view: "countries" as const,
            onView: vi.fn(),
        }
        const sheet = (setCountry = vi.fn()) => {
            useRace.setState({race: RACE})
            render(<BoardSheet country={france} setCountry={setCountry} leaderboard={[entry("fr", 500), entry("jp", 250)]}
                               tilesCount={1000} standings={standings} onCountryOrder={vi.fn()} onClose={vi.fn()}/>)
            return {setCountry, user: userEvent.setup()}
        }
        const head = () => document.querySelector(".sheet-head") as HTMLElement

        it("heads itself with the view picked, and keeps its name for a screen reader", () => {
            sheet()

            expect(within(head()).getByRole("button", {name: "Leaderboard: Countries"})).toBeDefined()
            const title = within(head()).getByRole("heading", {name: "Leaderboard"})
            expect(title.classList.contains("sr-only")).toBe(true)
            expect(document.querySelector(".sheet-body .board-heading")).toBeNull()
        })

        it("names the country played for on one line, with no box", () => {
            sheet()

            const line = screen.getByText("Playing for").closest(".menu-playing")!
            expect(line.classList.contains("panel-box")).toBe(false)
            expect(within(line as HTMLElement).getByText("France")).toBeDefined()
        })

        it("ranks the country on that line in the order picked", () => {
            useRace.setState({race: RACE})
            const board = (countryOrder: CountryOrder) => <BoardSheet country={france} setCountry={vi.fn()} tilesCount={1000}
                                                                      leaderboard={[entry("jp", 500), entry("fr", 250)]}
                                                                      standings={standings} countryOrder={countryOrder}
                                                                      onCountryOrder={vi.fn()} onClose={vi.fn()}/>

            const {rerender} = render(board("points"))
            expect(screen.getByLabelText("Rank 1").textContent).toBe("#1")

            rerender(board("tiles"))
            expect(screen.getByLabelText("Rank 2").textContent).toBe("#2")
        })

        it("changes the country from that line, and steps back to the board", async () => {
            const {setCountry, user} = sheet()

            await user.click(button("Change"))
            expect(within(head()).queryByRole("button", {name: /^Leaderboard:/})).toBeNull()
            expect(screen.getByRole("region", {name: "Change country"})).toBeDefined()

            await user.click(screen.getByRole("option", {name: /Japan/}))
            expect(setCountry).toHaveBeenCalledWith(Countries.get("jp"))
            expect(within(head()).getByRole("button", {name: "Leaderboard: Countries"})).toBeDefined()
        })
    })

    describe("the country picker", () => {
        it("opens on the Change button, not on the country name", async () => {
            const {user} = setup()

            await user.click(button("Change"))

            expect(countryPanel()).not.toBeNull()
        })

        it("replaces the place, and keeps the rail of places beside it", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))

            expect(board()).toBeNull()
            expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Board", "Share", "Settings"])
        })

        it("walks back up on the back arrow, and offers no other way out", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))
            expect(screen.queryByRole("button", {name: /close/i})).toBeNull()

            await user.click(button("Back"))

            expect(countryPanel()).toBeNull()
            expect(board()).not.toBeNull()
        })

        it("closes on Escape", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))
            await user.keyboard("{Escape}")

            expect(countryPanel()).toBeNull()
            expect(board()).not.toBeNull()
        })

        it("keeps the card header, and does not repeat it", async () => {
            const {user} = setup()

            await user.click(button("Change"))

            expect(screen.getByRole("heading", {name: "ClickPlanet", level: 1})).toBeDefined()
            expect(screen.getByRole("heading", {name: "Change country", level: 2})).toBeDefined()
        })

        it("exposes the panel as a labelled region, not a dialog", async () => {
            const {user, container} = setup()

            await user.click(button("Change"))

            expect(screen.getByRole("region", {name: "Change country"})).toBeDefined()
            expect(container.querySelector("[aria-modal]")).toBeNull()
        })

        it("moves focus into the panel, and hands it back to Change", async () => {
            const {user} = setup()

            await user.click(button("Change"))
            expect(document.activeElement)
                .toBe(screen.getByRole("heading", {name: "Change country", level: 2}))

            await user.click(button("Back"))
            expect(document.activeElement).toBe(button("Change"))
        })

        it("picks a country and returns to the leaderboard", async () => {
            const {user, setCountry} = setup([entry("fr", 500)])

            await user.click(button("Change"))
            await user.click(screen.getByRole("option", {name: "Japan"}))

            expect(setCountry).toHaveBeenCalledWith(Countries.get("jp"))
            expect(countryPanel()).toBeNull()
            expect(board()).not.toBeNull()
        })
    })

    describe("About", () => {
        const openAbout = async (user: ReturnType<typeof userEvent.setup>) => {
            await user.click(tab("Settings"))
            await user.click(button("About"))
        }

        it("opens as a modal dialog over the page, not inside the card", async () => {
            const {user, container} = setup()

            await openAbout(user)

            const dialog = aboutDialog()!
            expect(dialog).not.toBeNull()
            expect(dialog.getAttribute("aria-modal")).toBe("true")
            expect(container.querySelector(".menu")!.contains(dialog)).toBe(false)
        })

        it("pins the coffee button outside the scrolling copy", async () => {
            const {user} = setup()

            await openAbout(user)

            const coffee = screen.getByRole("link", {name: "Buy me a coffee"})
            expect(document.querySelector(".modal-footer")!.contains(coffee)).toBe(true)
            expect(document.querySelector(".modal-body")!.contains(coffee)).toBe(false)
        })

        it("closes on the ×, and hands focus back to the About button", async () => {
            const {user} = setup()

            await openAbout(user)
            await user.click(button("Close"))

            expect(aboutDialog()).toBeNull()
            expect(document.activeElement).toBe(button("About"))
        })

        it("closes on Escape", async () => {
            const {user} = setup()

            await openAbout(user)
            await user.keyboard("{Escape}")

            expect(aboutDialog()).toBeNull()
        })
    })

    const supportBackend = () => ({
        suggestFeature: vi.fn(async () => undefined),
        reportBug: vi.fn(async () => undefined),
        attachmentRules: vi.fn(async () => NO_ATTACHMENTS),
        uploadAttachment: vi.fn(async () => "file"),
    })

    const phone = {
        build: BUILD,
        viewportWidth: 390,
        viewportHeight: 844,
        pixelRatio: 3,
        connectionDown: true,
        gpuVendor: "Apple Inc.",
        gpuRenderer: "Apple GPU",
        webglVersion: "WebGL 2.0",
        maxTextureSize: 4096,
        contextLost: false,
        cores: 6,
        memoryGb: 4,
    }

    describe("feedback", () => {
        const withSupport = (pageContext?: () => typeof phone) => {
            const support = supportBackend()
            render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000} support={support}
                         pageContext={pageContext}/>)
            return {support, user: userEvent.setup()}
        }

        const write = async (user: ReturnType<typeof userEvent.setup>, kind: string, label: string, text: string) => {
            await user.click(tab("Settings"))
            await user.click(button("Feedback"))
            await user.click(screen.getByRole("radio", {name: kind}))
            await user.type(screen.getByRole("textbox", {name: label}), text)
        }

        it("is an idea written from Settings and sent with what the page knows about itself", async () => {
            const {support, user} = withSupport(() => phone)

            await write(user, "An idea", "Your idea", "a night mode")
            await user.click(button("Send"))

            expect(support.suggestFeature).toHaveBeenCalledWith("a night mode", phone, [])
            expect(support.reportBug).not.toHaveBeenCalled()
            expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
        })

        it("is a bug written from Settings and sent with what the page knows about itself", async () => {
            const {support, user} = withSupport(() => phone)

            await write(user, "A bug", "What happened", "the globe is black")
            await user.click(button("Send"))

            expect(support.reportBug).toHaveBeenCalledWith("the globe is black", phone, [])
            expect(support.suggestFeature).not.toHaveBeenCalled()
            expect(screen.getByRole("status").textContent).toBe("Sent. Thank you!")
        })

        it("reads the context when it is sent, not when the form opens", async () => {
            let connectionDown = false
            const {support, user} = withSupport(() => ({...phone, connectionDown}))

            await write(user, "A bug", "What happened", "the globe is black")
            connectionDown = true
            await user.click(button("Send"))

            expect(support.reportBug.mock.calls[0]).toMatchObject(["the globe is black", {connectionDown: true}, []])
        })

        it("says what the page alone knows when nothing tells it more", async () => {
            const {support, user} = withSupport()

            await write(user, "A bug", "What happened", "the globe is black")
            await user.click(button("Send"))

            expect(support.reportBug.mock.calls[0]).toMatchObject(["the globe is black", {
                build: BUILD,
                viewportWidth: window.innerWidth,
                viewportHeight: window.innerHeight,
                connectionDown: false,
                gpuRenderer: "",
            }, []])
        })
    })

    describe("Share", () => {
        it("takes the picture", async () => {
            const onTakePicture = vi.fn()
            render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000} onTakePicture={onTakePicture}/>)
            const user = userEvent.setup()

            await user.click(tab("Share"))
            await user.click(button("Take a picture"))

            expect(onTakePicture).toHaveBeenCalledTimes(1)
        })

        it("refuses a second press while a picture is being drawn, and keeps its label", async () => {
            render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000} onTakePicture={vi.fn()} taking/>)
            const user = userEvent.setup()

            await user.click(tab("Share"))

            const take = button("Take a picture")
            expect(take).toHaveProperty("disabled", true)
            expect(take.getAttribute("aria-busy")).toBe("true")
        })

        it("offers no picture when nothing can take one", async () => {
            const {user} = setup()

            await user.click(tab("Share"))

            expect(screen.queryByRole("button", {name: "Take a picture"})).toBeNull()
        })

        it("recruits for the country played, with its link", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(tab("Share"))

            expect(screen.getByText("Recruit for France")).toBeDefined()
            expect(screen.getByText("clickplanet.lol/?f=fr")).toBeDefined()
        })

        it("is CrazyGames' invite there, with no link that leads off it", async () => {
            render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000} onTakePicture={vi.fn()}
                         invite={vi.fn(async () => undefined)}/>)
            const user = userEvent.setup()

            expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Board", "Invite", "Settings"])
            await user.click(tab("Invite"))

            expect(button("Invite friends")).toBeDefined()
            expect(screen.queryByRole("button", {name: "Take a picture"})).toBeNull()
            expect(screen.queryByText("clickplanet.lol/?f=fr")).toBeNull()
        })
    })

    describe("the settings", () => {
        const withSettings = (display: DisplaySettings = DEFAULT_DISPLAY_SETTINGS) => {
            useDisplaySettings.setState(display)
            useSoundSettings.setState({settings: DEFAULT_SOUND_SETTINGS})
            const preview = vi.fn()
            const view = render(<Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                                      sound={{preview}}/>)
            return {...view, preview, user: userEvent.setup()}
        }
        const displayed = (): DisplaySettings => {
            const {mapView, rendering} = useDisplaySettings.getState()
            return {mapView, rendering}
        }

        const toggle = (name: string) => screen.getByRole("switch", {name}) as HTMLInputElement
        const flags = (name: string) => within(screen.getByRole("radiogroup", {name: "Country flags"})).getByRole("radio", {name})

        it("has the last tab, after Share", () => {
            withSettings()
            expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Board", "Share", "Settings"])
        })

        it("keeps the display and About with no sound to set", async () => {
            const {user} = setup()

            await user.click(tab("Settings"))

            expect(screen.getAllByRole("switch")).toEqual([toggle("HD graphics")])
            expect(button("About")).toBeDefined()
        })

        it("leaves the sound out of Share", async () => {
            const {user} = withSettings()
            await user.click(tab("Share"))
            expect(screen.queryByRole("switch")).toBeNull()
        })

        it("switches a sound off without playing it", async () => {
            const {user, preview} = withSettings()
            await user.click(tab("Settings"))

            await user.click(toggle("Chat message"))

            expect(useSoundSettings.getState().settings).toEqual({
                ...DEFAULT_SOUND_SETTINGS,
                sounds: {...DEFAULT_SOUND_SETTINGS.sounds, chat: false},
            })
            expect(preview).not.toHaveBeenCalled()
        })

        it("says what the display is set to", async () => {
            const {user} = withSettings({mapView: "tiles", rendering: "sharp"})
            await user.click(tab("Settings"))

            expect(flags("Off").getAttribute("aria-checked")).toBe("true")
            expect(flags("Split").getAttribute("aria-checked")).toBe("false")
            expect(toggle("HD graphics").checked).toBe(true)
        })

        it("picks one of the three ways to draw the flags", async () => {
            const {user} = withSettings()
            await user.click(tab("Settings"))
            expect(flags("Split").getAttribute("aria-checked")).toBe("true")

            for (const [label, mapView] of [["Big", "big"], ["Off", "tiles"], ["Split", "split"]]) {
                await user.click(flags(label))
                expect(displayed()).toEqual({...DEFAULT_DISPLAY_SETTINGS, mapView})
                expect(flags(label).getAttribute("aria-checked")).toBe("true")
            }
        })

        it("turns the HD graphics off, keeping the map view", async () => {
            const {user} = withSettings({mapView: "tiles", rendering: "sharp"})
            await user.click(tab("Settings"))
            await user.click(toggle("HD graphics"))
            expect(displayed()).toEqual({mapView: "tiles", rendering: "soft"})
        })
    })

    describe("the account", () => {
        const settler = {id: "settler", name: "Settler", rank: {trackId: "conquest", trackName: "Conquest", number: 1, count: 5}}
        const raider = {id: "raider", name: "Raider", rank: {trackId: "conquest", trackName: "Conquest", number: 2, count: 5}}
        const og = {id: "og", name: "OG"}
        const dashboard = (progress: number): TitleDashboard => ({
            worn: settler,
            wearable: [og, settler],
            tracks: [{
                id: "conquest", name: "Conquest", progress, steps: [
                    {title: settler, threshold: 100, earned: true},
                    {title: raider, threshold: 1_000, earned: false},
                ],
            }],
        })

        const withAccount = (offered: Provider[], me: Me, username = "", linkedMultiplier?: number, playerInfo = NO_PLAYER, clicks?: AcceptedClicks, phone = false) => {
            const backend = {
                signInOptions: vi.fn(async () => offered),
                me: vi.fn(async () => me),
                startSignIn: vi.fn(async () => "https://google.example/authorize"),
                completeSignIn: vi.fn(async () => "signedIn" as const),
                startEmailSignIn: vi.fn(async () => undefined),
                completeEmailSignIn: vi.fn(async () => "signedIn" as const),
                completeCrazyGamesSignIn: vi.fn(async () => "signedIn" as const),
                signOut: vi.fn(async () => undefined),
                signOutEverywhere: vi.fn(async () => undefined),
                deleteAccount: vi.fn(async () => undefined),
            } satisfies AccountBackend
            const navigate = vi.fn()
            const player = {
                profile: vi.fn(async () => ({accountId: "account-1", name: username, color: NameColor.UNSPECIFIED})),
                setName: vi.fn(async (name: string) => ({accountId: "account-1", name})),
                setColor: vi.fn(async (color: NameColor) => color),
            } satisfies PlayerBackend
            const progression = {
                titles: vi.fn(async (): Promise<TitleDashboard> => ({wearable: [], tracks: []})),
                fronts: vi.fn(async (): Promise<Fronts> => ({playsFor: [], playsAgainst: []})),
                wearTitle: vi.fn(async (): Promise<PlayerTitle | undefined> => undefined),
            } satisfies ProgressionBackend
            const store = new AccountStore(backend, player, progression, {token: vi.fn(), held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn()}, {navigate, remember: vi.fn(), claimFlag: vi.fn()})
            const view = render(phone
                ? <YouPlace account={store} playerInfo={playerInfo} listenForClicks={clicks?.listenForClicks} clips={new FakeClipsBackend()}/>
                : <Menu {...backends()} country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                        account={store} linkedMultiplier={linkedMultiplier} playerInfo={playerInfo}
                        listenForClicks={clicks?.listenForClicks}/>)
            const user = userEvent.setup()
            const openSettings = async () => {
                await user.click(await screen.findByRole("button", {name: "You"}))
            }
            return {...view, backend, player, progression, navigate, user, openSettings}
        }

        it("puts the account at the foot of the rail, out of the places", async () => {
            withAccount(["google"], {linked: ["google"]}, "ana")

            const you = await screen.findByRole("button", {name: "You"})
            expect(you.getAttribute("aria-haspopup")).toBe("dialog")
            expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Board", "Share", "Settings"])
        })

        it("opens a signed-in player's progress and account side by side, in one dialog", async () => {
            const {user, progression} = withAccount(["google"], {linked: ["google"]}, "ana")
            progression.titles.mockResolvedValue(dashboard(140))

            await user.click(await screen.findByRole("button", {name: "You"}))

            const profile = screen.getByRole("dialog", {name: "ana"})
            expect(within(profile).getByRole("region", {name: "Account"})).toBeDefined()
            expect(within(profile).getByRole("tablist", {name: "Title tracks"})).toBeDefined()
            expect(await within(profile).findByText("860 tiles to Raider")).toBeDefined()
            expect(within(profile).getByLabelText("Username")).toBeDefined()
            expect(screen.queryByRole("tablist", {name: "Account"})).toBeNull()
        })

        it("offers no sign-in without an account store", () => {
            setup()
            expect(screen.queryByRole("button", {name: "Sign in"})).toBeNull()
        })

        it("offers no sign-in while no provider is offered", async () => {
            const {backend} = withAccount([], {linked: []})

            await vi.waitFor(() => expect(backend.me).toHaveBeenCalled())

            expect(screen.queryByRole("button", {name: "Sign in"})).toBeNull()
        })

        it("offers only the providers the server offers, with the privacy policy", async () => {
            const {user} = withAccount(["google"], {linked: []})

            await user.click(await screen.findByRole("button", {name: "Sign in"}))

            expect(screen.getByRole("button", {name: "Sign in with Google"})).toBeDefined()
            expect(screen.queryByRole("button", {name: "Sign in with Discord"})).toBeNull()
            expect(screen.getByRole("link", {name: "Privacy policy"}).getAttribute("href")).toBe("/privacy")
        })

        it("tells a guest how much faster a signed-in player clicks, as the server said", async () => {
            const {user} = withAccount(["google"], {linked: []}, "", 2)

            await user.click(await screen.findByRole("button", {name: "Sign in"}))

            expect(screen.getByText(/Sign in to click 2× faster/)).toBeDefined()
            expect(screen.getByText(/You do not need an account to play/)).toBeDefined()
        })

        it("promises no speed a server did not report", async () => {
            const {user} = withAccount(["google"], {linked: []})

            await user.click(await screen.findByRole("button", {name: "Sign in"}))

            expect(screen.queryByText(/faster/)).toBeNull()
        })

        it("leaves for the provider", async () => {
            const {user, navigate, backend} = withAccount(["google"], {linked: []})

            await user.click(await screen.findByRole("button", {name: "Sign in"}))
            await user.click(button("Sign in with Google"))

            expect(backend.startSignIn).toHaveBeenCalledWith("google", "signIn")
            expect(navigate).toHaveBeenCalledWith("https://google.example/authorize")
        })

        it("sends a link, not a sign-in, from a linked account", async () => {
            const {user, backend, openSettings} = withAccount(["google", "discord"], {linked: ["discord"]})

            await openSettings()
            await user.click(button("Link Google"))

            expect(backend.startSignIn).toHaveBeenCalledWith("google", "link")
        })

        it("shows who is signed in, and links the missing provider", async () => {
            const {openSettings} = withAccount(["google", "discord"], {linked: ["google"]})

            await openSettings()

            expect(within(screen.getByRole("list", {name: "Signed in with"})).getAllByRole("listitem")[0].textContent).toBe("Google")
            expect(button("Link Discord")).toBeDefined()
            expect(screen.queryByRole("button", {name: "Link Google"})).toBeNull()
            expect(button("Sign out")).toBeDefined()
            expect(button("Sign out everywhere")).toBeDefined()
        })

        it("opens a signed-in player's account on its progress, read again each time the account opens", async () => {
            const {user, progression} = withAccount(["google"], {linked: ["google"]}, "ana")
            progression.titles.mockResolvedValue(dashboard(140))

            await user.click(await screen.findByRole("button", {name: "You"}))
            expect(await screen.findByText("860 tiles to Raider")).toBeDefined()

            progression.titles.mockResolvedValue(dashboard(400))
            await user.click(button("Close"))
            await user.click(button("You"))
            expect(await screen.findByText("600 tiles to Raider")).toBeDefined()
            expect(progression.titles).toHaveBeenCalledTimes(2)
        })

        it("shows a signed-in player its own stats on its progress", async () => {
            const playerInfo = {
                playerInfo: vi.fn(async () => ({
                    name: "ana", guest: false, tilesTaken: 14_212, streakCurrent: 31, streakBest: 31, admin: false,
                    color: NameColor.UNSPECIFIED, titles: [], playsFor: [], playsAgainst: [],
                })),
            }
            const {user, progression} = withAccount(["google"], {linked: ["google"]}, "ana", undefined, playerInfo)
            progression.titles.mockResolvedValue(dashboard(140))

            await user.click(await screen.findByRole("button", {name: "You"}))

            expect(await screen.findByText("14,212")).toBeDefined()
            expect(playerInfo.playerInfo).toHaveBeenCalledWith("account-1")
        })

        it("counts the tiles a player takes into its stats at once, and reads its titles again once its clicks settle", async () => {
            const playerInfo = {
                playerInfo: vi.fn(async () => ({
                    name: "ana", guest: false, tilesTaken: 14_212, streakCurrent: 31, streakBest: 31, admin: false,
                    color: NameColor.UNSPECIFIED, titles: [], playsFor: [], playsAgainst: [],
                })),
            }
            const clicks = acceptedClicks()
            const {user, progression} = withAccount(["google"], {linked: ["google"]}, "ana", undefined, playerInfo, clicks)
            progression.titles.mockResolvedValue(dashboard(140))
            await user.click(await screen.findByRole("button", {name: "You"}))
            expect(await screen.findByText("860 tiles to Raider")).toBeDefined()
            expect(await screen.findByText("14,212")).toBeDefined()

            vi.useFakeTimers()
            progression.titles.mockResolvedValue(dashboard(143))
            act(() => {
                clicks.record({country: "fr", took: true})
                clicks.record({country: "de", took: true})
                clicks.record({country: "fr", took: false})
            })
            expect(screen.getByText("14,214")).toBeDefined()
            expect(progression.titles).toHaveBeenCalledTimes(1)

            await act(async () => vi.advanceTimersByTime(SETTLE_MS))
            vi.useRealTimers()

            expect(progression.titles).toHaveBeenCalledTimes(2)
            expect(await screen.findByText("857 tiles to Raider")).toBeDefined()
            expect(playerInfo.playerInfo).toHaveBeenCalledTimes(1)
        })

        it("shows the first countries a signed-in player fought for and against, the rest on demand, and reads them again once its clicks settle", async () => {
            const clicks = acceptedClicks()
            const {user, progression} = withAccount(["google"], {linked: ["google"]}, "ana", undefined, undefined, clicks)
            progression.titles.mockResolvedValue(dashboard(140))
            const against = ["de", "es", "it", "pt", "be"].map((countryCode, i) => ({countryCode, tiles: 50 - i}))
            progression.fronts.mockResolvedValue({playsFor: [{countryCode: "fr", tiles: 500}], playsAgainst: against})

            await user.click(await screen.findByRole("button", {name: "You"}))

            const foughtAgainst = await screen.findByRole("region", {name: "Fought against"})
            const foughtFor = screen.getByRole("region", {name: "Fought for"})
            expect(within(foughtAgainst).getAllByRole("listitem").map((row) => row.textContent))
                .toEqual(["Germany50", "Spain49", "Italy48"])
            expect(within(foughtFor).getAllByRole("listitem")).toHaveLength(1)
            expect(within(foughtFor).queryByRole("button")).toBeNull()

            await user.click(within(foughtAgainst).getByRole("button", {name: "See all 5"}))
            expect(within(foughtAgainst).getAllByRole("listitem")).toHaveLength(5)

            vi.useFakeTimers()
            progression.fronts.mockResolvedValue({playsFor: [{countryCode: "fr", tiles: 501}], playsAgainst: against})
            act(() => clicks.record({country: "fr", took: true}))
            await act(async () => vi.advanceTimersByTime(SETTLE_MS))
            vi.useRealTimers()

            expect(progression.fronts).toHaveBeenCalledTimes(2)
            expect(await screen.findByText("501")).toBeDefined()
        })

        it("folds the countries fought for and against in a phone's narrow sheet", async () => {
            const {user, progression} = withAccount(["google"], {linked: ["google"]}, "ana", undefined, undefined, undefined, true)
            progression.titles.mockResolvedValue(dashboard(140))
            const against = ["de", "es", "it", "pt", "be"].map((countryCode, i) => ({countryCode, tiles: 50 - i}))
            progression.fronts.mockResolvedValue({playsFor: [{countryCode: "fr", tiles: 500}], playsAgainst: against})

            const foughtAgainst = await screen.findByRole("region", {name: "Fought against"})
            expect(within(foughtAgainst).queryByRole("list")).toBeNull()
            expect(screen.getByRole("button", {name: "Titles"}).getAttribute("aria-expanded")).toBe("true")

            await user.click(within(foughtAgainst).getByRole("button", {name: "Fought against, 5 countries"}))
            expect(within(foughtAgainst).getAllByRole("listitem")).toHaveLength(5)
        })

        it("lists the tracks, and every rank of the first one beside them", async () => {
            const {user, progression} = withAccount(["google"], {linked: ["google"]}, "ana")
            progression.titles.mockResolvedValue(dashboard(140))

            await user.click(await screen.findByRole("button", {name: "You"}))
            const conquest = await screen.findByRole("tab", {name: /Conquest · rank 1 of 2/})
            expect(conquest.getAttribute("aria-selected")).toBe("true")

            const ranks = screen.getByRole("tabpanel", {name: /Conquest/})
            expect(within(ranks).getAllByRole("listitem")).toHaveLength(2)
        })

        it("wears the title pressed, and shows it worn", async () => {
            const {user, progression} = withAccount(["google"], {linked: ["google"]}, "ana")
            progression.titles.mockResolvedValueOnce(dashboard(140)).mockResolvedValue({...dashboard(140), worn: og})
            progression.wearTitle.mockResolvedValue(og)

            await user.click(await screen.findByRole("button", {name: "You"}))
            const wear = await screen.findByRole("radiogroup", {name: "Wear a title"})
            expect(within(wear).getByRole("radio", {name: "Settler"}).getAttribute("aria-checked")).toBe("true")

            await user.click(within(wear).getByRole("radio", {name: "OG"}))

            expect(progression.wearTitle).toHaveBeenCalledWith("og")
            await vi.waitFor(() =>
                expect(within(wear).getByRole("radio", {name: "OG"}).getAttribute("aria-checked")).toBe("true"))
        })

        it("shows a guest no tabs, and reads no titles", async () => {
            const {user, progression} = withAccount(["google"], {linked: []})

            await user.click(await screen.findByRole("button", {name: "Sign in"}))

            expect(screen.queryByRole("tablist", {name: "Account"})).toBeNull()
            expect(progression.titles).not.toHaveBeenCalled()
        })

        it("shows the username, and saves a new one", async () => {
            const {user, player, openSettings} = withAccount(["google"], {linked: ["google"]}, "ana")

            await openSettings()
            const input = await screen.findByDisplayValue("ana")
            expect(button("Save")).toHaveProperty("disabled", true)

            await user.clear(input)
            await user.type(input, "bo")
            expect(button("Save")).toHaveProperty("disabled", true)

            await user.type(input, "b")
            await user.click(button("Save"))

            expect(player.setName).toHaveBeenCalledWith("bob")
            expect(await screen.findAllByText("bob")).toHaveLength(3)
        })

        it("offers a color to a player with a username, and saves the one pressed", async () => {
            const {user, player, openSettings} = withAccount(["google"], {linked: ["google"]}, "ana")
            await openSettings()

            const colors = await screen.findByRole("group", {name: "Name color"})
            expect(within(colors).getAllByRole("button")).toHaveLength(12)
            expect(within(colors).getAllByRole("button").filter((b) => b.getAttribute("aria-pressed") === "true")).toHaveLength(0)

            await user.click(within(colors).getByRole("button", {name: "Teal"}))

            expect(player.setColor).toHaveBeenCalledWith(NameColor.TEAL)
            await vi.waitFor(() =>
                expect(within(colors).getByRole("button", {name: "Teal"}).getAttribute("aria-pressed")).toBe("true"))
        })

        it("offers no color before a username is chosen", async () => {
            const {openSettings} = withAccount(["google"], {linked: ["google"]})
            await openSettings()
            await screen.findByLabelText("Username")

            expect(screen.queryByRole("group", {name: "Name color"})).toBeNull()
        })

        it("says why a username was refused", async () => {
            const {user, player, openSettings} = withAccount(["google"], {linked: ["google"]})
            player.setName.mockRejectedValue(new PlayerError("taken"))

            await openSettings()
            await user.type(screen.getByLabelText("Username"), "ana")
            await user.click(button("Save"))

            expect((await screen.findByRole("alert")).textContent).toBe("Another player has this username.")
        })

        it("offers no username to a guest", async () => {
            const {user} = withAccount(["google"], {linked: []})

            await user.click(await screen.findByRole("button", {name: "Sign in"}))

            expect(screen.queryByLabelText("Username")).toBeNull()
        })

        it("deletes the account only after the dialog says what goes", async () => {
            const {user, backend, openSettings} = withAccount(["google"], {linked: ["google"]})

            await openSettings()
            await user.click(button("Delete account"))

            const dialog = screen.getByRole("dialog", {name: "Delete your account?"})
            expect(within(dialog).getByText(/within 30 days/)).toBeDefined()
            expect(backend.deleteAccount).not.toHaveBeenCalled()

            await user.click(within(dialog).getByRole("button", {name: "Delete"}))

            await vi.waitFor(() => expect(screen.queryByRole("dialog", {name: "Delete your account?"})).toBeNull())
            expect(backend.deleteAccount).toHaveBeenCalledTimes(1)
            expect(button("Sign in with Google")).toBeDefined()
        })

        it("keeps the account when the dialog is cancelled", async () => {
            const {user, backend, openSettings} = withAccount(["google"], {linked: ["google"]})

            await openSettings()
            await user.click(button("Delete account"))
            await user.click(button("Cancel"))

            expect(screen.queryByRole("dialog", {name: "Delete your account?"})).toBeNull()
            expect(button("Delete account")).toBeDefined()
            expect(backend.deleteAccount).not.toHaveBeenCalled()
        })
    })
})
