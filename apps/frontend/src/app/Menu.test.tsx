// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import Menu from "./Menu.tsx"
import {Countries} from "../domain/countries.ts"
import type {LeaderboardEntry} from "../domain/leaderboard.ts"
import {DEFAULT_SOUND_SETTINGS} from "../domain/soundSettings.ts"
import {AccountBackend, Me, Provider} from "../backends/account.ts"
import {AccountStore} from "./account/accountStore.ts"
import {RosterEntry} from "../backends/player.ts"
import {PlayerBackend, PlayerError} from "../backends/player.ts"

const france = Countries.get("fr")!
const entry = (code: string, tiles: number) => ({country: Countries.get(code)!, tiles})

afterEach(cleanup)

function setup(leaderboard: LeaderboardEntry[] = [], country = france) {
    const setCountry = vi.fn()
    const view = render(
        <Menu country={country} setCountry={setCountry} leaderboard={leaderboard} tilesCount={1000}/>,
    )
    return {...view, setCountry, user: userEvent.setup()}
}

const leaderboardRows = () => screen.queryAllByRole("row").slice(1)
const countryPanel = () => screen.queryByRole("listbox", {name: "Country"})
const aboutDialog = () => screen.queryByRole("dialog", {name: "About ClickPlanet"})
const button = (name: string | RegExp) => screen.getByRole("button", {name})
const collapse = () => button("ClickPlanet menu")

describe("Menu", () => {
    it("starts on the leaderboard, with the picker and About closed", () => {
        setup([entry("fr", 500)])

        expect(leaderboardRows()).toHaveLength(1)
        expect(countryPanel()).toBeNull()
        expect(aboutDialog()).toBeNull()
    })

    it("draws the country you play for with a flag, not an emoji", () => {
        const {container} = setup([entry("fr", 500)])

        const playing = container.querySelector(".menu-playing-name")!
        expect(playing.querySelectorAll(".country-flag")).toHaveLength(1)
        expect(playing.textContent).toBe("France")
    })

    it("links Home to the home page, which does not send the player back", () => {
        setup()
        expect(screen.getByRole("link", {name: "Home"}).getAttribute("href")).toBe("/#home")
    })

    describe("the collapse", () => {
        it("folds the card away, keeping the country and its rank on screen", async () => {
            const {user} = setup([entry("jp", 500), entry("fr", 250)])

            await user.click(collapse())

            expect(leaderboardRows()).toHaveLength(0)
            expect(screen.queryByRole("button", {name: "About"})).toBeNull()
            expect(screen.getByText("France")).toBeDefined()
            expect(screen.getByText("#2")).toBeDefined()
        })

        it("keeps the folded country's flag an image, not an emoji", async () => {
            const {user, container} = setup([entry("fr", 250)])

            await user.click(collapse())

            const folded = container.querySelector(".menu-header-country")!
            expect(folded.querySelectorAll(".country-flag")).toHaveLength(1)
            expect(folded.textContent).not.toMatch(/\p{RI}|\p{Extended_Pictographic}/u)
        })

        it("unfolds it again", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(collapse())
            await user.click(collapse())

            expect(leaderboardRows()).toHaveLength(1)
        })

        it("says whether it is open, and what it controls", async () => {
            const {user, container} = setup()

            expect(collapse().getAttribute("aria-expanded")).toBe("true")
            const controlled = collapse().getAttribute("aria-controls")!
            expect(container.querySelector(`[id="${controlled}"]`)).not.toBeNull()

            await user.click(collapse())
            expect(collapse().getAttribute("aria-expanded")).toBe("false")
            expect(collapse().getAttribute("aria-controls")).toBeNull()
        })

        it("shows a dash rather than a rank when the country holds no tile", () => {
            setup([entry("jp", 500)])
            expect(screen.getByText("—")).toBeDefined()
        })

        it("opens unfolded when nothing says the viewport is compact", () => {
            setup([entry("fr", 500)])
            expect(leaderboardRows()).toHaveLength(1)
        })
    })

    describe("the country picker", () => {
        it("opens on the Change button, not on the country name", async () => {
            const {user} = setup()

            await user.click(button("Change"))

            expect(countryPanel()).not.toBeNull()
        })

        it("replaces the leaderboard and the buttons", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))

            expect(leaderboardRows()).toHaveLength(0)
            expect(screen.queryByRole("button", {name: "About"})).toBeNull()
        })

        it("walks back up on the back arrow, and offers no other way out", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))
            expect(screen.queryByRole("button", {name: /close/i})).toBeNull()

            await user.click(button("Back"))

            expect(countryPanel()).toBeNull()
            expect(leaderboardRows()).toHaveLength(1)
        })

        it("closes on Escape", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))
            await user.keyboard("{Escape}")

            expect(countryPanel()).toBeNull()
            expect(leaderboardRows()).toHaveLength(1)
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
            expect(leaderboardRows()).toHaveLength(1)
        })
    })

    describe("About", () => {
        it("opens as a modal dialog over the page, not inside the card", async () => {
            const {user, container} = setup()

            await user.click(button("About"))

            const dialog = aboutDialog()!
            expect(dialog).not.toBeNull()
            expect(dialog.getAttribute("aria-modal")).toBe("true")
            expect(container.querySelector(".menu")!.contains(dialog)).toBe(false)
        })

        it("leaves the leaderboard standing behind it", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("About"))

            const table = screen.getByRole("region", {name: "Leaderboard"})
            expect(within(table).getByText("France")).toBeDefined()
        })

        it("pins the coffee button outside the scrolling copy", async () => {
            const {user, container} = setup()

            await user.click(button("About"))

            const coffee = screen.getByRole("link", {name: "Buy me a coffee"})
            expect(container.querySelector(".modal-footer")!.contains(coffee)).toBe(true)
            expect(container.querySelector(".modal-body")!.contains(coffee)).toBe(false)
        })

        it("closes on the ×, and hands focus back to the About button", async () => {
            const {user} = setup()

            await user.click(button("About"))
            await user.click(button("Close"))

            expect(aboutDialog()).toBeNull()
            expect(document.activeElement).toBe(button("About"))
        })

        it("closes on Escape", async () => {
            const {user} = setup()

            await user.click(button("About"))
            await user.keyboard("{Escape}")

            expect(aboutDialog()).toBeNull()
        })
    })

    describe("the sound settings", () => {
        const withSound = () => {
            const onChange = vi.fn()
            const preview = vi.fn()
            const view = render(<Menu country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                                      sound={{settings: DEFAULT_SOUND_SETTINGS, onChange, preview}}/>)
            return {...view, onChange, preview, user: userEvent.setup()}
        }

        it("offers no sound button without sound settings", () => {
            setup()
            expect(screen.queryByRole("button", {name: "Sound settings"})).toBeNull()
        })

        it("switches a sound off without playing it", async () => {
            const {user, onChange, preview} = withSound()
            await user.click(button("Sound settings"))

            await user.click(screen.getByRole("switch", {name: "Chat message"}))

            expect(onChange).toHaveBeenCalledWith({
                ...DEFAULT_SOUND_SETTINGS,
                sounds: {...DEFAULT_SOUND_SETTINGS.sounds, chat: false},
            })
            expect(preview).not.toHaveBeenCalled()
        })

        it("goes back to the button that opened it", async () => {
            const {user} = withSound()
            await user.click(button("Sound settings"))
            await user.click(button("Back"))

            expect(document.activeElement).toBe(button("Sound settings"))
        })
    })

    describe("the players", () => {
        const players = [
            {name: "ana", tag: "4f2ca1", countryCode: "fr", guest: false},
            {name: "guest_Bo", tag: "91aa3d", countryCode: "de", guest: true},
        ]
        const withPlayers = (entries = players) => ({
            ...render(<Menu country={france} setCountry={vi.fn()} leaderboard={[entry("fr", 500)]} tilesCount={1000}
                            players={entries}/>),
            user: userEvent.setup(),
        })

        // No roster wired, or a server without one.
        it("offers no list without a roster", () => {
            setup()
            expect(screen.queryByRole("button", {name: /online/})).toBeNull()
        })

        it("says how many are playing on the button", () => {
            withPlayers()

            const players = button("2 players online")
            expect(players.textContent).toBe("2")
        })

        it("counts one player in the singular", () => {
            withPlayers(players.slice(0, 1))
            expect(button("1 player online")).toBeDefined()
        })

        it("opens the list in place of the leaderboard, and goes back to the button", async () => {
            const {user} = withPlayers()

            await user.click(button("2 players online"))

            expect(screen.getByRole("region", {name: "Players online"})).toBeDefined()
            expect(leaderboardRows()).toHaveLength(0)
            expect(screen.getByText("ana")).toBeDefined()

            await user.click(button("Back"))

            expect(leaderboardRows()).toHaveLength(1)
            expect(document.activeElement).toBe(button("2 players online"))
        })

        it("shows the button, and an empty list, when nobody is playing", async () => {
            const {user} = withPlayers([])

            await user.click(button("0 players online"))

            expect(screen.getByText("Nobody is playing right now.")).toBeDefined()
        })
    })

    describe("the account", () => {
        const withAccount = (offered: Provider[], me: Me, username = "", linkedMultiplier?: number, players?: RosterEntry[]) => {
            const backend = {
                signInOptions: vi.fn(async () => offered),
                me: vi.fn(async () => me),
                startSignIn: vi.fn(async () => "https://google.example/authorize"),
                completeSignIn: vi.fn(async () => undefined),
                signOut: vi.fn(async () => undefined),
                signOutEverywhere: vi.fn(async () => undefined),
                deleteAccount: vi.fn(async () => undefined),
            } satisfies AccountBackend
            const navigate = vi.fn()
            const player = {
                profile: vi.fn(async () => ({accountId: "account-1", name: username})),
                setName: vi.fn(async (name: string) => ({accountId: "account-1", name})),
            } satisfies PlayerBackend
            const store = new AccountStore(backend, player, {token: vi.fn(), held: vi.fn(), invalidate: vi.fn()}, {navigate, remember: vi.fn()})
            const view = render(<Menu country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                                      account={store} linkedMultiplier={linkedMultiplier} players={players}/>)
            return {...view, backend, player, navigate, user: userEvent.setup()}
        }

        it("keeps the account button beside the players button", async () => {
            withAccount(["google"], {linked: ["google"]}, "ana", undefined,
                [{name: "ana", tag: "4f2ca1", countryCode: "fr", guest: false}])

            expect(await screen.findByRole("button", {name: "Account"})).toBeDefined()
            expect(screen.getByRole("button", {name: "1 player online"})).toBeDefined()
        })

        it("offers no sign-in without an account store", () => {
            setup()
            expect(screen.queryByRole("button", {name: "Sign in"})).toBeNull()
        })

        // Production runs with sign-in off: the menu must look as it did.
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

        // The production bug: a link sent as a sign-in moved the player to the other account.
        it("sends a link, not a sign-in, from a linked account", async () => {
            const {user, backend} = withAccount(["google", "discord"], {linked: ["discord"]})

            await user.click(await screen.findByRole("button", {name: "Account"}))
            await user.click(button("Link Google"))

            expect(backend.startSignIn).toHaveBeenCalledWith("google", "link")
        })

        it("shows who is signed in, and links the missing provider", async () => {
            const {user} = withAccount(["google", "discord"], {linked: ["google"]})

            await user.click(await screen.findByRole("button", {name: "Account"}))

            expect(screen.getByText("Signed in with Google.")).toBeDefined()
            expect(button("Link Discord")).toBeDefined()
            expect(screen.queryByRole("button", {name: "Link Google"})).toBeNull()
            expect(button("Sign out")).toBeDefined()
            expect(button("Sign out everywhere")).toBeDefined()
        })

        it("shows the username, and saves a new one", async () => {
            const {user, player} = withAccount(["google"], {linked: ["google"]}, "ana")

            await user.click(await screen.findByRole("button", {name: "Account"}))
            const input = await screen.findByDisplayValue("ana")
            expect(button("Save")).toHaveProperty("disabled", true)

            await user.clear(input)
            await user.type(input, "bo")
            expect(button("Save")).toHaveProperty("disabled", true)

            await user.type(input, "b")
            await user.click(button("Save"))

            expect(player.setName).toHaveBeenCalledWith("bob")
            expect(await screen.findByText("bob")).toBeDefined()
        })

        it("says why a username was refused", async () => {
            const {user, player} = withAccount(["google"], {linked: ["google"]})
            player.setName.mockRejectedValue(new PlayerError("taken"))

            await user.click(await screen.findByRole("button", {name: "Account"}))
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
            const {user, backend} = withAccount(["google"], {linked: ["google"]})

            await user.click(await screen.findByRole("button", {name: "Account"}))
            await user.click(button("Delete account"))

            const dialog = screen.getByRole("dialog", {name: "Delete your account?"})
            expect(within(dialog).getByText(/cannot undo/)).toBeDefined()
            expect(backend.deleteAccount).not.toHaveBeenCalled()

            await user.click(within(dialog).getByRole("button", {name: "Delete"}))

            await vi.waitFor(() => expect(screen.queryByRole("dialog")).toBeNull())
            expect(backend.deleteAccount).toHaveBeenCalledTimes(1)
            expect(button("Sign in with Google")).toBeDefined()
        })

        it("keeps the account when the dialog is cancelled", async () => {
            const {user, backend} = withAccount(["google"], {linked: ["google"]})

            await user.click(await screen.findByRole("button", {name: "Account"}))
            await user.click(button("Delete account"))
            await user.click(button("Cancel"))

            expect(screen.queryByRole("dialog")).toBeNull()
            expect(backend.deleteAccount).not.toHaveBeenCalled()
        })
    })
})
