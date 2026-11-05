import {describe, expect, it, vi} from "vitest"
import {connectCrazyGames, Sdk} from "./crazyGamesSdk.ts"
import {InviteParams} from "../domain/crazyGames.ts"
import {Countries} from "../domain/countries.ts"

type FakeOverrides = {
    environment?: Sdk["environment"]
    token?: () => Promise<string>
    prompt?: () => Promise<unknown>
    inviteParams?: InviteParams
}

function fakeSdk(overrides: FakeOverrides = {}) {
    const authListeners = new Set<(user: unknown) => void>()
    const settingsListeners = new Set<(settings: Sdk["game"]["settings"]) => void>()
    const joinListeners = new Set<(params: InviteParams) => void>()
    const sdk = {
        init: vi.fn(async () => undefined),
        environment: overrides.environment ?? "crazygames",
        user: {
            isUserAccountAvailable: true,
            getUserToken: vi.fn(overrides.token ?? (async () => "the-jwt")),
            showAuthPrompt: vi.fn(overrides.prompt ?? (async () => ({username: "Nova"}))),
            addAuthListener: (listener: (user: unknown) => void) => authListeners.add(listener),
            removeAuthListener: (listener: (user: unknown) => void) => authListeners.delete(listener),
        },
        game: {
            settings: {disableChat: false, muteAudio: false} as Sdk["game"]["settings"],
            addSettingsChangeListener: (listener: (settings: Sdk["game"]["settings"]) => void) => settingsListeners.add(listener),
            removeSettingsChangeListener: (listener: (settings: Sdk["game"]["settings"]) => void) => settingsListeners.delete(listener),
            loadingStart: vi.fn(),
            loadingStop: vi.fn(),
            gameplayStart: vi.fn(),
            inviteParams: overrides.inviteParams ?? null,
            updateRoom: vi.fn(),
            inviteLink: vi.fn((params: InviteParams) => `https://www.crazygames.com/game/clickplanet?czy_invite=true&f=${params.f}`),
            addJoinRoomListener: (listener: (params: InviteParams) => void) => joinListeners.add(listener),
            removeJoinRoomListener: (listener: (params: InviteParams) => void) => joinListeners.delete(listener),
        },
    } satisfies Sdk
    return {
        sdk,
        signIn: (user: unknown) => authListeners.forEach((listener) => listener(user)),
        change: (settings: Sdk["game"]["settings"]) => settingsListeners.forEach((listener) => listener(settings)),
        join: (params: InviteParams) => joinListeners.forEach((listener) => listener(params)),
        authListeners,
        joinListeners,
    }
}

const settle = () => new Promise((resolve) => setTimeout(resolve, 0))

describe("connectCrazyGames", () => {
    it("asks for the in-game token, and a guest has none", async () => {
        const signedIn = fakeSdk()
        expect(await connectCrazyGames(async () => signedIn.sdk).user.token()).toBe("the-jwt")
        expect(signedIn.sdk.user.getUserToken).toHaveBeenCalledWith({signingKeyType: "IN_GAME_AUTH"})

        const guest = fakeSdk({token: () => Promise.reject({code: "userNotAuthenticated"})})
        expect(await connectCrazyGames(async () => guest.sdk).user.token()).toBeUndefined()
    })

    it("is a guest when the SDK cannot load, or runs off CrazyGames", async () => {
        const errors = vi.spyOn(console, "error").mockImplementation(() => undefined)
        for (const page of [
            connectCrazyGames(() => Promise.reject(new Error("blocked"))),
            connectCrazyGames(async () => fakeSdk({environment: "disabled"}).sdk),
        ]) {
            expect(await page.user.token()).toBeUndefined()
            expect(await page.user.prompt()).toBe(false)
            expect(page.settings()).toEqual({chat: true, audio: true})
        }
        errors.mockRestore()
    })

    it("says whether the player signed in at the prompt", async () => {
        expect(await connectCrazyGames(async () => fakeSdk().sdk).user.prompt()).toBe(true)
        expect(await connectCrazyGames(async () => fakeSdk({prompt: () => Promise.reject({code: "userCancelled"})}).sdk).user.prompt()).toBe(false)
        expect(await connectCrazyGames(async () => fakeSdk({prompt: () => Promise.reject({code: "userAlreadySignedIn"})}).sdk).user.prompt()).toBe(true)
    })

    it("tells a sign-in, and stops when asked", async () => {
        const fake = fakeSdk()
        const heard = vi.fn()
        const stop = connectCrazyGames(async () => fake.sdk).user.listenForSignIn(heard)
        await settle()

        fake.signIn(null)
        fake.signIn({username: "Nova"})
        expect(heard).toHaveBeenCalledTimes(1)

        stop()
        expect(fake.authListeners.size).toBe(0)
    })

    it("reads the settings, then each change", async () => {
        const fake = fakeSdk()
        fake.sdk.game.settings = {disableChat: true, muteAudio: false}
        const page = connectCrazyGames(async () => fake.sdk)
        const heard = vi.fn()

        page.listenForSettings(heard)
        await settle()
        fake.change({disableChat: false, muteAudio: true})

        expect(heard).toHaveBeenNthCalledWith(1, {chat: false, audio: true})
        expect(heard).toHaveBeenNthCalledWith(2, {chat: true, audio: false})
        expect(page.settings()).toEqual({chat: false, audio: true})
    })

    it("tells the loading, then the gameplay once the game starts", async () => {
        const fake = fakeSdk()
        const page = connectCrazyGames(async () => fake.sdk)
        await settle()
        expect(fake.sdk.game.loadingStart).toHaveBeenCalledTimes(1)
        expect(fake.sdk.game.gameplayStart).not.toHaveBeenCalled()

        page.started()
        await settle()

        expect(fake.sdk.game.loadingStop).toHaveBeenCalledTimes(1)
        expect(fake.sdk.game.gameplayStart).toHaveBeenCalledTimes(1)
    })

    it("reports the one world, open, with the flag played for", async () => {
        const fake = fakeSdk()
        connectCrazyGames(async () => fake.sdk).room.report("fr")
        await settle()

        expect(fake.sdk.game.updateRoom).toHaveBeenCalledWith({roomId: "world", isJoinable: true, inviteParams: {f: "fr"}})
    })

    it("keeps playing when CrazyGames refuses a room report", async () => {
        const errors = vi.spyOn(console, "error").mockImplementation(() => undefined)
        const fake = fakeSdk()
        fake.sdk.game.updateRoom.mockImplementation(() => {
            throw {code: "rateLimited"}
        })

        connectCrazyGames(async () => fake.sdk).room.report("fr")
        await settle()

        expect(errors).toHaveBeenCalledTimes(1)
        errors.mockRestore()
    })

    it("tells the invite the page opened from once, then each invite taken while it plays", async () => {
        const fake = fakeSdk({inviteParams: {f: "fr", czy_invited_by: "someone"}})
        const page = connectCrazyGames(async () => fake.sdk)
        const first = vi.fn()
        const second = vi.fn()

        const stop = page.room.listenForInvites(first)
        page.room.listenForInvites(second)
        await settle()
        fake.join({f: "de"})
        fake.join({roomName: "lobby"})

        expect(first.mock.calls).toEqual([[Countries.get("fr")], [Countries.get("de")]])
        expect(second.mock.calls).toEqual([[Countries.get("de")]])

        stop()
        expect(fake.joinListeners.size).toBe(1)
    })

    it("tells the opening invite to the listener that stays after a mount, an unmount and a mount", async () => {
        const fake = fakeSdk({inviteParams: {f: "fr"}})
        const page = connectCrazyGames(async () => fake.sdk)
        const gone = vi.fn()
        const stays = vi.fn()

        page.room.listenForInvites(gone)()
        page.room.listenForInvites(stays)
        await settle()

        expect(gone).not.toHaveBeenCalled()
        expect(stays).toHaveBeenCalledWith(Countries.get("fr"))
    })

    it("asks CrazyGames for a link that invites to the flag, and has none off CrazyGames", async () => {
        const fake = fakeSdk()
        expect(await connectCrazyGames(async () => fake.sdk).room.inviteLink("fr"))
            .toBe("https://www.crazygames.com/game/clickplanet?czy_invite=true&f=fr")
        expect(fake.sdk.game.inviteLink).toHaveBeenCalledWith({f: "fr"})

        expect(await connectCrazyGames(async () => fakeSdk({environment: "disabled"}).sdk).room.inviteLink("fr")).toBeUndefined()
    })
})
