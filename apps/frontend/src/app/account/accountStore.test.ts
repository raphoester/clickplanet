import {describe, expect, it, vi} from "vitest"
import {AccountBackend, AuthError, AuthFailure, Me, Provider} from "../../backends/account.ts"
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

const refusing = (failure: AuthFailure) => async () => {
    throw new AuthError(failure)
}

function setup(backend: Fake) {
    const session: SessionProvider = {token: vi.fn(async () => "token"), invalidate: vi.fn()}
    const navigate = vi.fn()
    const remember = vi.fn()
    const store = new AccountStore(backend as unknown as AccountBackend, session, {navigate, remember})
    return {store, session, navigate, remember}
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
})
