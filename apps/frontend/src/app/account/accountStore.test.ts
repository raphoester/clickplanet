import {describe, expect, it, vi} from "vitest"
import {AccountBackend, AuthError, AuthFailure, Me, Provider} from "../../backends/account.ts"
import {PlayerBackend, PlayerError, PlayerFailure, Profile} from "../../backends/player.ts"
import {SessionProvider} from "../../backends/session.ts"
import {AccountStore} from "./accountStore.ts"

type Fake = {[K in keyof AccountBackend]: ReturnType<typeof vi.fn>}

function fakeBackend(offered: Provider[] = ["google", "discord"], me: Me = {linked: []}): Fake {
    return {
        signInOptions: vi.fn(async () => offered),
        me: vi.fn(async () => me),
        startSignIn: vi.fn(async (provider: Provider) => `https://${provider}.example/authorize`),
        completeSignIn: vi.fn(async () => undefined),
        signOut: vi.fn(async () => undefined),
        signOutEverywhere: vi.fn(async () => undefined),
        deleteAccount: vi.fn(async () => undefined),
    }
}

type FakePlayer = {[K in keyof PlayerBackend]: ReturnType<typeof vi.fn>}

function fakePlayer(name = ""): FakePlayer {
    return {
        profile: vi.fn(async (): Promise<Profile> => ({accountId: "account-1", name})),
        setName: vi.fn(async (name: string): Promise<Profile> => ({accountId: "account-1", name})),
    }
}

const refusing = (failure: AuthFailure) => async () => {
    throw new AuthError(failure)
}

const refusingName = (failure: PlayerFailure) => async () => {
    throw new PlayerError(failure)
}

function setup(backend: Fake, player: FakePlayer = fakePlayer()) {
    const session: SessionProvider = {token: vi.fn(async () => "token"), invalidate: vi.fn()}
    const navigate = vi.fn()
    const remember = vi.fn()
    const store = new AccountStore(
        backend as unknown as AccountBackend, player as unknown as PlayerBackend, session, {navigate, remember})
    return {store, session, navigate, remember, player}
}

/** Resolves a promise the test holds back until it calls `release`. */
function held<T>() {
    let release: (value: T) => void = () => {}
    const promise = new Promise<T>((resolve) => {
        release = resolve
    })
    return {promise, release}
}

describe("AccountStore", () => {
    describe("loading", () => {
        it("starts loading, then shows the offered providers in a fixed order", async () => {
            const {store} = setup(fakeBackend(["discord", "google"]))
            expect(store.state()).toEqual({kind: "loading"})

            await store.load()

            expect(store.state()).toEqual({kind: "ready", offered: ["google", "discord"], me: {linked: []}})
        })

        it("hides everything while sign-in is off", async () => {
            const {store} = setup(fakeBackend([]))

            await store.load()

            expect(store.state()).toEqual({kind: "hidden"})
        })

        // Login is optional: a section that cannot say what it offers is better absent.
        it("hides everything when the options cannot be read", async () => {
            const backend = fakeBackend()
            backend.signInOptions.mockImplementation(refusing("failed"))
            const {store} = setup(backend)

            await store.load()

            expect(store.state()).toEqual({kind: "hidden"})
        })

        // Sign-in turned off on the server must not strand a player who can still sign out.
        it("still shows a linked account when no provider is offered", async () => {
            const {store} = setup(fakeBackend([], {linked: ["google"]}))

            await store.load()

            expect(store.state()).toEqual({kind: "ready", offered: [], me: {linked: ["google"]}})
        })

        it("shows a guest when the account cannot be read", async () => {
            const backend = fakeBackend()
            backend.me.mockImplementation(refusing("failed"))
            const {store} = setup(backend)

            await store.load()

            expect(store.state()).toMatchObject({kind: "ready", me: {linked: []}})
        })

        it("shares one read between two callers at once", async () => {
            const backend = fakeBackend()
            const {store} = setup(backend)

            await Promise.all([store.load(), store.load()])

            expect(backend.signInOptions).toHaveBeenCalledTimes(1)
            expect(backend.me).toHaveBeenCalledTimes(1)
        })

        it("tells its listeners", async () => {
            const {store} = setup(fakeBackend())
            const listener = vi.fn()
            store.subscribe(listener)

            await store.load()

            expect(listener).toHaveBeenCalled()
        })
    })

    describe("signing in", () => {
        it("remembers the provider and leaves for its page", async () => {
            const backend = fakeBackend()
            const {store, navigate, remember} = setup(backend)
            await store.load()

            await store.signIn("discord")

            expect(backend.startSignIn).toHaveBeenCalledWith("discord", "signIn")
            expect(remember).toHaveBeenCalledWith("discord", "signIn")
            expect(navigate).toHaveBeenCalledWith("https://discord.example/authorize")
        })

        // A link never moves the browser to another account: the server refuses instead.
        it("links from a linked account, and remembers it is a link", async () => {
            const backend = fakeBackend(["google", "discord"], {linked: ["discord"]})
            const {store, navigate, remember} = setup(backend)
            await store.load()

            await store.link("google")

            expect(backend.startSignIn).toHaveBeenCalledWith("google", "link")
            expect(remember).toHaveBeenCalledWith("google", "link")
            expect(navigate).toHaveBeenCalledWith("https://google.example/authorize")
        })

        it("shows a guest when a link finds the account gone", async () => {
            const backend = fakeBackend(["google", "discord"], {linked: ["discord"]})
            backend.startSignIn.mockImplementation(refusing("notSignedIn"))
            const {store} = setup(backend)
            await store.load()

            await store.link("google")

            expect(store.state()).toEqual({
                kind: "ready", offered: ["google", "discord"], me: {linked: []}, failure: "notSignedIn",
            })
        })

        it("leaves from the callback page without a load, and throws there", async () => {
            const backend = fakeBackend()
            const {store, navigate} = setup(backend)

            await store.leaveFor("google", "link")
            expect(backend.startSignIn).toHaveBeenCalledWith("google", "link")
            expect(navigate).toHaveBeenCalledWith("https://google.example/authorize")

            backend.startSignIn.mockImplementation(refusing("tooManyTries"))
            await expect(store.leaveFor("google", "signIn")).rejects.toMatchObject({failure: "tooManyTries"})
        })

        it("is busy while the server answers, and starts nothing else meanwhile", async () => {
            const backend = fakeBackend()
            let release = () => {}
            backend.startSignIn.mockImplementation(() => new Promise<string>((resolve) => {
                release = () => resolve("https://google.example")
            }))
            const {store} = setup(backend)
            await store.load()

            const signingIn = store.signIn("google")
            expect(store.state()).toMatchObject({busy: "signIn"})

            await store.signOut()
            expect(backend.signOut).not.toHaveBeenCalled()

            release()
            await signingIn
        })

        it("does nothing before the load", async () => {
            const backend = fakeBackend()
            const {store} = setup(backend)

            await store.signIn("google")

            expect(backend.startSignIn).not.toHaveBeenCalled()
        })

        // Sign-in shares the mint budget. The player waits and presses again.
        it("reports a spent budget and stays where it was", async () => {
            const backend = fakeBackend()
            backend.startSignIn.mockImplementation(refusing("tooManyTries"))
            const {store, navigate} = setup(backend)
            await store.load()

            await store.signIn("google")

            expect(navigate).not.toHaveBeenCalled()
            expect(store.state()).toEqual({
                kind: "ready", offered: ["google", "discord"], me: {linked: []}, failure: "tooManyTries",
            })
        })

        it("takes away a button the server says it does not offer", async () => {
            const backend = fakeBackend()
            backend.startSignIn.mockImplementation(refusing("notOffered"))
            const {store} = setup(backend)
            await store.load()

            await store.signIn("google")

            expect(store.state()).toMatchObject({offered: ["discord"], failure: "notOffered"})
        })

        it("hides the section when the server turned sign-in off since the load", async () => {
            const backend = fakeBackend()
            backend.startSignIn.mockImplementation(refusing("off"))
            const {store} = setup(backend)
            await store.load()

            await store.signIn("google")

            expect(store.state()).toEqual({kind: "hidden"})
        })

        it("clears the last failure when the next action starts", async () => {
            const backend = fakeBackend()
            backend.startSignIn.mockImplementationOnce(refusing("tooManyTries"))
            const {store} = setup(backend)
            await store.load()
            await store.signIn("google")

            await store.signIn("google")

            expect(store.state()).not.toHaveProperty("failure")
        })
    })

    describe("completing a sign-in", () => {
        it("invalidates the click token, then reads the account again", async () => {
            const backend = fakeBackend()
            const {store, session} = setup(backend)
            backend.me.mockImplementation(async () => ({linked: ["google"]}))

            await store.completeSignIn("code", "state")

            expect(backend.completeSignIn).toHaveBeenCalledWith("code", "state")
            expect(session.invalidate).toHaveBeenCalledTimes(1)
            expect(store.state()).toMatchObject({kind: "ready", me: {linked: ["google"]}})
        })

        it("throws a refusal for the callback page and keeps the token", async () => {
            const backend = fakeBackend()
            backend.completeSignIn.mockImplementation(refusing("startAgain"))
            const {store, session} = setup(backend)

            await expect(store.completeSignIn("code", "state")).rejects.toMatchObject({failure: "startAgain"})
            expect(session.invalidate).not.toHaveBeenCalled()
        })
    })

    describe("leaving the account", () => {
        const actions = ["signOut", "signOutEverywhere", "deleteAccount"] as const

        for (const action of actions) {
            it(`${action} leaves a guest and invalidates the click token`, async () => {
                const backend = fakeBackend(["google", "discord"], {linked: ["google"]})
                const {store, session} = setup(backend)
                await store.load()

                await store[action]()

                expect(backend[action]).toHaveBeenCalledTimes(1)
                expect(session.invalidate).toHaveBeenCalledTimes(1)
                expect(store.state()).toEqual({kind: "ready", offered: ["google", "discord"], me: {linked: []}})
            })

            it(`${action} forgets the username`, async () => {
                const {store} = setup(fakeBackend(["google"], {linked: ["google"]}), fakePlayer("ana"))
                await store.load()
                await vi.waitFor(() => expect(store.state()).toMatchObject({username: "ana"}))

                await store[action]()

                expect(store.state()).not.toHaveProperty("username", "ana")
            })
        }

        it("treats an account already gone as done", async () => {
            const backend = fakeBackend(["google"], {linked: ["google"]})
            backend.deleteAccount.mockImplementation(refusing("notSignedIn"))
            const {store, session} = setup(backend)
            await store.load()

            await store.deleteAccount()

            expect(session.invalidate).toHaveBeenCalledTimes(1)
            expect(store.state()).toEqual({kind: "ready", offered: ["google"], me: {linked: []}})
        })

        it("keeps the account and the token when the server fails", async () => {
            const backend = fakeBackend(["google"], {linked: ["google"]})
            backend.deleteAccount.mockImplementation(refusing("failed"))
            const {store, session} = setup(backend)
            await store.load()

            await store.deleteAccount()

            expect(session.invalidate).not.toHaveBeenCalled()
            expect(store.state()).toEqual({kind: "ready", offered: ["google"], me: {linked: ["google"]}, failure: "failed"})
        })

        it("hides the section after signing out when sign-in is off", async () => {
            const {store} = setup(fakeBackend([], {linked: ["discord"]}))
            await store.load()

            await store.signOut()

            expect(store.state()).toEqual({kind: "hidden"})
        })
    })
    describe("the username", () => {
        const linked = () => fakeBackend(["google"], {linked: ["google"]})

        it("is read once the account shows a linked provider", async () => {
            const {store, player} = setup(linked(), fakePlayer("ana"))

            await store.load()

            await vi.waitFor(() => expect(store.state()).toEqual({
                kind: "ready", offered: ["google"], me: {linked: ["google"]}, username: "ana",
            }))
            expect(player.profile).toHaveBeenCalledTimes(1)
        })

        // Reading it needs a click token: a guest must not mint just to learn it has none.
        it("is not read for a guest", async () => {
            const {store, player} = setup(fakeBackend(), fakePlayer("ana"))

            await store.load()

            expect(player.profile).not.toHaveBeenCalled()
            expect(store.state()).not.toHaveProperty("username", "ana")
        })

        it("shows the section before the read answers", async () => {
            const player = fakePlayer()
            const profile = held<Profile>()
            player.profile.mockReturnValue(profile.promise)
            const {store} = setup(linked(), player)

            await store.load()
            expect(store.state()).toMatchObject({kind: "ready", me: {linked: ["google"]}})

            profile.release({accountId: "account-1", name: "ana"})
            await vi.waitFor(() => expect(store.state()).toMatchObject({username: "ana"}))
        })

        it("is left unknown when the read fails, and the section still shows", async () => {
            const player = fakePlayer()
            player.profile.mockImplementation(refusingName("failed"))
            const {store} = setup(linked(), player)

            await store.load()
            await vi.waitFor(() => expect(player.profile).toHaveBeenCalled())

            expect(store.state()).toEqual({kind: "ready", offered: ["google"], me: {linked: ["google"]}})
        })

        it("is none while the player has not chosen one", async () => {
            const {store, player} = setup(linked(), fakePlayer(""))

            await store.load()
            await vi.waitFor(() => expect(player.profile).toHaveBeenCalled())

            expect(store.state()).toEqual({kind: "ready", offered: ["google"], me: {linked: ["google"]}})
        })

        it("drops a read that lands after the player signed out", async () => {
            const player = fakePlayer()
            const profile = held<Profile>()
            player.profile.mockReturnValue(profile.promise)
            const {store} = setup(linked(), player)
            await store.load()

            await store.signOut()
            profile.release({accountId: "account-1", name: "ana"})
            await profile.promise

            expect(store.state()).toEqual({kind: "ready", offered: ["google"], me: {linked: []}})
        })

        it("is read again after a sign-in completes", async () => {
            const backend = fakeBackend()
            backend.me.mockImplementation(async () => ({linked: ["google"]}))
            const {store, player} = setup(backend, fakePlayer("ana"))

            await store.completeSignIn("code", "state")

            await vi.waitFor(() => expect(store.state()).toMatchObject({username: "ana"}))
            expect(player.profile).toHaveBeenCalledTimes(1)
        })

        it("is kept when another action fails", async () => {
            const backend = linked()
            backend.signOut.mockImplementation(refusing("failed"))
            const {store} = setup(backend, fakePlayer("ana"))
            await store.load()
            await vi.waitFor(() => expect(store.state()).toMatchObject({username: "ana"}))

            await store.signOut()

            expect(store.state()).toMatchObject({username: "ana", failure: "failed"})
        })

        describe("saving it", () => {
            it("is busy while the server answers, then shows the name it stored", async () => {
                const player = fakePlayer()
                const saved = held<Profile>()
                player.setName.mockReturnValue(saved.promise)
                const {store} = setup(linked(), player)
                await store.load()

                const saving = store.setUsername("Ana")
                expect(player.setName).toHaveBeenCalledWith("Ana")
                expect(store.state()).toMatchObject({naming: true})

                saved.release({accountId: "account-1", name: "Ana"})
                await saving

                expect(store.state()).toEqual({
                    kind: "ready", offered: ["google"], me: {linked: ["google"]}, username: "Ana",
                })
            })

            it("reports why it was refused and keeps the name it had", async () => {
                const player = fakePlayer("ana")
                player.setName.mockImplementation(refusingName("taken"))
                const {store} = setup(linked(), player)
                await store.load()
                await vi.waitFor(() => expect(store.state()).toMatchObject({username: "ana"}))

                await store.setUsername("bo")

                expect(store.state()).toEqual({
                    kind: "ready", offered: ["google"], me: {linked: ["google"]}, username: "ana", nameFailure: "taken",
                })
            })

            it("clears the last refusal when the next save starts", async () => {
                const player = fakePlayer()
                player.setName.mockImplementationOnce(refusingName("invalid"))
                const {store} = setup(linked(), player)
                await store.load()
                await store.setUsername("bo")

                const saving = store.setUsername("bob")
                expect(store.state()).not.toHaveProperty("nameFailure")
                await saving
            })

            it("does nothing for a guest", async () => {
                const {store, player} = setup(fakeBackend())
                await store.load()

                await store.setUsername("ana")

                expect(player.setName).not.toHaveBeenCalled()
            })

            it("does nothing before the load", async () => {
                const {store, player} = setup(linked())

                await store.setUsername("ana")

                expect(player.setName).not.toHaveBeenCalled()
            })

            it("does not start while another action runs", async () => {
                const backend = linked()
                const signingOut = held<undefined>()
                backend.signOut.mockReturnValue(signingOut.promise)
                const {store, player} = setup(backend)
                await store.load()

                const out = store.signOut()
                await store.setUsername("ana")

                expect(player.setName).not.toHaveBeenCalled()
                signingOut.release(undefined)
                await out
            })

            it("holds every other action back while it runs", async () => {
                const backend = linked()
                const player = fakePlayer()
                const saved = held<Profile>()
                player.setName.mockReturnValue(saved.promise)
                const {store} = setup(backend, player)
                await store.load()

                const saving = store.setUsername("ana")
                await store.signOut()

                expect(backend.signOut).not.toHaveBeenCalled()
                saved.release({accountId: "account-1", name: "ana"})
                await saving
            })

            it("drops an answer that lands after the account was read again", async () => {
                const backend = linked()
                const player = fakePlayer()
                const saved = held<Profile>()
                player.setName.mockReturnValue(saved.promise)
                const {store} = setup(backend, player)
                await store.load()

                const saving = store.setUsername("ana")
                backend.me.mockImplementation(async () => ({linked: []}))
                await store.completeSignIn("code", "state")
                saved.release({accountId: "account-1", name: "ana"})
                await saving

                expect(store.state()).toEqual({kind: "ready", offered: ["google"], me: {linked: []}})
            })
        })
    })
})
