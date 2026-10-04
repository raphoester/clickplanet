// @vitest-environment jsdom
import {ReactNode} from "react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen, within} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import Menu from "./Menu.tsx"
import {Countries} from "../domain/countries.ts"
import type {LeaderboardEntry} from "../domain/leaderboard.ts"
import {DEFAULT_SOUND_SETTINGS} from "../domain/soundSettings.ts"
import {AccountBackend, Me, Provider} from "../backends/account.ts"
import {AccountStore} from "./account/accountStore.ts"
import {DISCORD_INVITE} from "../links.ts"
import {NameColor} from "../backends/player.ts"
import {PlayerBackend, PlayerError, PlayerInfoBackend, PlayerTitle, TitleDashboard} from "../backends/player.ts"

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

const board = () => screen.queryByRole("region", {name: "Leaderboard"})
const countryPanel = () => screen.queryByRole("listbox", {name: "Country"})
const aboutDialog = () => screen.queryByRole("dialog", {name: "About ClickPlanet"})
const button = (name: string | RegExp) => screen.getByRole("button", {name})
const tab = (name: string) => screen.getByRole("tab", {name})

describe("Menu", () => {
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

    it("offers the board and More, and the account only when there is one", () => {
        setup()
        expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Board", "More"])
    })

    it("swaps the board for the place picked, in the same panel", async () => {
        const {user} = setup([entry("fr", 500)])

        await user.click(tab("More"))

        expect(tab("More").getAttribute("aria-selected")).toBe("true")
        expect(board()).toBeNull()
        expect(button("About")).toBeDefined()
    })

    it("links Home to the home page, which does not send the player back", async () => {
        const {user} = setup()
        await user.click(tab("More"))
        expect(screen.getByRole("link", {name: "Home page"}).getAttribute("href")).toBe("/#home")
    })

    it("links Discord to the server's invite, in a new tab", async () => {
        const {user} = setup()
        await user.click(tab("More"))

        const discord = screen.getByRole("link", {name: "Discord"})
        expect(discord.getAttribute("href")).toBe(DISCORD_INVITE)
        expect(discord.getAttribute("target")).toBe("_blank")
        expect(discord.getAttribute("rel")).toBe("noopener noreferrer")
    })

    describe("the leader", () => {
        const toll = [{share: 0.1, slowdown: 1.5}, {share: 0.3, slowdown: 4}]
        const withLeader = (data: LeaderboardEntry[], anthem?: ReactNode) => render(
            <Menu country={france} setCountry={vi.fn()} leaderboard={data} tilesCount={1000} toll={toll} anthem={anthem}/>)

        it("is drawn in a frame of its own, apart from the table", () => {
            withLeader([entry("jp", 500), entry("fr", 250)])

            const leader = screen.getByRole("region", {name: "First: Japan"})
            expect(within(leader).getByText("50.00")).toBeDefined()
            expect(screen.getAllByRole("row").slice(1).map((r) => within(r).getAllByRole("cell")[1].textContent))
                .toEqual(["France"])
        })

        it("says how much slower it refills, from the toll", () => {
            withLeader([entry("jp", 500)])
            expect(within(screen.getByRole("region", {name: "First: Japan"})).getByText("Refills 4× slower")).toBeDefined()
        })

        it("says nothing about the toll under its first step", () => {
            withLeader([entry("jp", 50)])
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
            backend: {standings: vi.fn(async () => []), mySeason: vi.fn(async () => undefined)},
            caller: {linked: false},
            listenForClicks: () => () => {},
            view: "countries" as const,
            onView: vi.fn(),
        }

        it("are offered beside the countries once wired, and the countries stay as they were", () => {
            render(<Menu country={france} setCountry={vi.fn()} leaderboard={[entry("fr", 500)]} tilesCount={1000} standings={standings}/>)

            expect(within(screen.getByRole("tablist", {name: "Leaderboard"})).getAllByRole("tab").map((t) => t.textContent))
                .toEqual(["Countries", "Players", "France"])
            expect(screen.getByRole("region", {name: "First: France"})).toBeDefined()
        })

        it("are not offered when none are wired", () => {
            setup([entry("fr", 500)])
            expect(screen.queryByRole("tablist", {name: "Leaderboard"})).toBeNull()
        })
    })

    describe("the country picker", () => {
        it("opens on the Change button, not on the country name", async () => {
            const {user} = setup()

            await user.click(button("Change"))

            expect(countryPanel()).not.toBeNull()
        })

        it("replaces the places and their tabs", async () => {
            const {user} = setup([entry("fr", 500)])

            await user.click(button("Change"))

            expect(board()).toBeNull()
            expect(screen.queryByRole("tab")).toBeNull()
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
            await user.click(tab("More"))
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

    describe("the picture", () => {
        it("is taken from More", async () => {
            const onTakePicture = vi.fn()
            render(<Menu country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000} onTakePicture={onTakePicture}/>)
            const user = userEvent.setup()

            await user.click(tab("More"))
            await user.click(button("Take a picture"))

            expect(onTakePicture).toHaveBeenCalledTimes(1)
        })

        it("refuses a second press while one is being drawn, and keeps its label", async () => {
            render(<Menu country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000} onTakePicture={vi.fn()} taking/>)
            const user = userEvent.setup()

            await user.click(tab("More"))

            const take = button("Take a picture")
            expect(take).toHaveProperty("disabled", true)
            expect(take.getAttribute("aria-busy")).toBe("true")
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

        it("offers no sound button without sound settings", async () => {
            const {user} = setup()
            await user.click(tab("More"))
            expect(screen.queryByRole("button", {name: "Sound"})).toBeNull()
        })

        it("switches a sound off without playing it", async () => {
            const {user, onChange, preview} = withSound()
            await user.click(tab("More"))
            await user.click(button("Sound"))

            await user.click(screen.getByRole("switch", {name: "Chat message"}))

            expect(onChange).toHaveBeenCalledWith({
                ...DEFAULT_SOUND_SETTINGS,
                sounds: {...DEFAULT_SOUND_SETTINGS.sounds, chat: false},
            })
            expect(preview).not.toHaveBeenCalled()
        })

        it("goes back to the button that opened it", async () => {
            const {user} = withSound()
            await user.click(tab("More"))
            await user.click(button("Sound"))
            await user.click(button("Back"))

            expect(document.activeElement).toBe(button("Sound"))
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

        const withAccount = (offered: Provider[], me: Me, username = "", linkedMultiplier?: number, playerInfo?: PlayerInfoBackend) => {
            const backend = {
                signInOptions: vi.fn(async () => offered),
                me: vi.fn(async () => me),
                startSignIn: vi.fn(async () => "https://google.example/authorize"),
                completeSignIn: vi.fn(async () => undefined),
                startEmailSignIn: vi.fn(async () => undefined),
                completeEmailSignIn: vi.fn(async () => undefined),
                signOut: vi.fn(async () => undefined),
                signOutEverywhere: vi.fn(async () => undefined),
                deleteAccount: vi.fn(async () => undefined),
            } satisfies AccountBackend
            const navigate = vi.fn()
            const player = {
                profile: vi.fn(async () => ({accountId: "account-1", name: username, color: NameColor.UNSPECIFIED})),
                setName: vi.fn(async (name: string) => ({accountId: "account-1", name})),
                setColor: vi.fn(async (color: NameColor) => color),
                titles: vi.fn(async (): Promise<TitleDashboard> => ({wearable: [], tracks: []})),
                wearTitle: vi.fn(async (): Promise<PlayerTitle | undefined> => undefined),
            } satisfies PlayerBackend
            const store = new AccountStore(backend, player, {token: vi.fn(), held: vi.fn(), identity: vi.fn(), heldIdentity: vi.fn(), invalidate: vi.fn()}, {navigate, remember: vi.fn()})
            const view = render(<Menu country={france} setCountry={vi.fn()} leaderboard={[]} tilesCount={1000}
                                      account={store} linkedMultiplier={linkedMultiplier} playerInfo={playerInfo}/>)
            const user = userEvent.setup()
            const openSettings = async () => {
                await user.click(await screen.findByRole("tab", {name: "You"}))
                await user.click(screen.getByRole("tab", {name: "Settings"}))
            }
            return {...view, backend, player, navigate, user, openSettings}
        }

        it("puts the account between the board and More", async () => {
            withAccount(["google"], {linked: ["google"]}, "ana")

            await screen.findByRole("tab", {name: "You"})
            expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Board", "You", "More"])
        })

        it("offers no sign-in without an account store", () => {
            setup()
            expect(screen.queryByRole("tab", {name: "Sign in"})).toBeNull()
        })

        it("offers no sign-in while no provider is offered", async () => {
            const {backend} = withAccount([], {linked: []})

            await vi.waitFor(() => expect(backend.me).toHaveBeenCalled())

            expect(screen.queryByRole("tab", {name: "Sign in"})).toBeNull()
        })

        it("offers only the providers the server offers, with the privacy policy", async () => {
            const {user} = withAccount(["google"], {linked: []})

            await user.click(await screen.findByRole("tab", {name: "Sign in"}))

            expect(screen.getByRole("button", {name: "Sign in with Google"})).toBeDefined()
            expect(screen.queryByRole("button", {name: "Sign in with Discord"})).toBeNull()
            expect(screen.getByRole("link", {name: "Privacy policy"}).getAttribute("href")).toBe("/privacy")
        })

        it("tells a guest how much faster a signed-in player clicks, as the server said", async () => {
            const {user} = withAccount(["google"], {linked: []}, "", 2)

            await user.click(await screen.findByRole("tab", {name: "Sign in"}))

            expect(screen.getByText(/Sign in to click 2× faster/)).toBeDefined()
            expect(screen.getByText(/You do not need an account to play/)).toBeDefined()
        })

        it("promises no speed a server did not report", async () => {
            const {user} = withAccount(["google"], {linked: []})

            await user.click(await screen.findByRole("tab", {name: "Sign in"}))

            expect(screen.queryByText(/faster/)).toBeNull()
        })

        it("leaves for the provider", async () => {
            const {user, navigate, backend} = withAccount(["google"], {linked: []})

            await user.click(await screen.findByRole("tab", {name: "Sign in"}))
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

            expect(screen.getByText("Signed in with Google.")).toBeDefined()
            expect(button("Link Discord")).toBeDefined()
            expect(screen.queryByRole("button", {name: "Link Google"})).toBeNull()
            expect(button("Sign out")).toBeDefined()
            expect(button("Sign out everywhere")).toBeDefined()
        })

        it("opens a signed-in player's account on its progress, read again each time the account opens", async () => {
            const {user, player} = withAccount(["google"], {linked: ["google"]}, "ana")
            player.titles.mockResolvedValue(dashboard(140))

            await user.click(await screen.findByRole("tab", {name: "You"}))
            expect(screen.getByRole("tab", {name: "Progress"}).getAttribute("aria-selected")).toBe("true")
            expect(await screen.findByText("860 tiles to Raider")).toBeDefined()

            player.titles.mockResolvedValue(dashboard(400))
            await user.click(screen.getByRole("tab", {name: "Board"}))
            await user.click(screen.getByRole("tab", {name: "You"}))
            expect(await screen.findByText("600 tiles to Raider")).toBeDefined()
            expect(player.titles).toHaveBeenCalledTimes(2)
        })

        it("shows a signed-in player its own stats on its progress", async () => {
            const playerInfo = {
                playerInfo: vi.fn(async (name: string) => ({
                    name, tilesTaken: 14_212, streakCurrent: 31, streakBest: 31, admin: false,
                    color: NameColor.UNSPECIFIED, titles: [],
                })),
            }
            const {user, player} = withAccount(["google"], {linked: ["google"]}, "ana", undefined, playerInfo)
            player.titles.mockResolvedValue(dashboard(140))

            await user.click(await screen.findByRole("tab", {name: "You"}))

            expect(await screen.findByText("14,212")).toBeDefined()
            expect(playerInfo.playerInfo).toHaveBeenCalledWith("ana")
        })

        it("wears the title pressed, and shows it worn", async () => {
            const {user, player} = withAccount(["google"], {linked: ["google"]}, "ana")
            player.titles.mockResolvedValueOnce(dashboard(140)).mockResolvedValue({...dashboard(140), worn: og})
            player.wearTitle.mockResolvedValue(og)

            await user.click(await screen.findByRole("tab", {name: "You"}))
            const wear = await screen.findByRole("radiogroup", {name: "Wear a title"})
            expect(within(wear).getByRole("radio", {name: "Settler"}).getAttribute("aria-checked")).toBe("true")

            await user.click(within(wear).getByRole("radio", {name: "OG"}))

            expect(player.wearTitle).toHaveBeenCalledWith("og")
            await vi.waitFor(() =>
                expect(within(wear).getByRole("radio", {name: "OG"}).getAttribute("aria-checked")).toBe("true"))
        })

        it("shows a guest no tabs, and reads no titles", async () => {
            const {user, player} = withAccount(["google"], {linked: []})

            await user.click(await screen.findByRole("tab", {name: "Sign in"}))

            expect(screen.queryByRole("tablist", {name: "Account"})).toBeNull()
            expect(player.titles).not.toHaveBeenCalled()
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

            await user.click(await screen.findByRole("tab", {name: "Sign in"}))

            expect(screen.queryByLabelText("Username")).toBeNull()
        })

        it("deletes the account only after the dialog says what goes", async () => {
            const {user, backend, openSettings} = withAccount(["google"], {linked: ["google"]})

            await openSettings()
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
            const {user, backend, openSettings} = withAccount(["google"], {linked: ["google"]})

            await openSettings()
            await user.click(button("Delete account"))
            await user.click(button("Cancel"))

            expect(screen.queryByRole("dialog")).toBeNull()
            expect(backend.deleteAccount).not.toHaveBeenCalled()
        })
    })
})
