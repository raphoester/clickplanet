// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {FakeMarketingBackend} from "../../backends/fakeMarketingBackend.ts"
import {SEASON_EMAILS_CONSENT} from "../../domain/seasonEmailsConsent.ts"
import SeasonEmails from "./SeasonEmails.tsx"
import {SeasonEmailsStore} from "./seasonEmailsStore.ts"
import {useSyncExternalStore} from "react"

afterEach(cleanup)

function Live({store, onSignIn}: {store: SeasonEmailsStore, onSignIn?: () => void}) {
    const state = useSyncExternalStore(store.subscribe, store.state)
    return <SeasonEmails state={state} store={store} onSignIn={onSignIn}/>
}

async function signedIn(backend = new FakeMarketingBackend(["ada@example.com"])) {
    const store = new SeasonEmailsStore(backend)
    await store.follow({linked: ["google"]})
    render(<Live store={store}/>)
    return {store, backend}
}

describe("SeasonEmails", () => {
    it("says the consent in the words the server keeps as its version", async () => {
        await signedIn()

        expect(screen.getByRole("button", {name: "Email me before the finale"}).textContent).toBe(SEASON_EMAILS_CONSENT.words)
    })

    it("sends a guest to the sign-in pitch", async () => {
        const store = new SeasonEmailsStore(new FakeMarketingBackend())
        await store.follow({linked: []})
        const onSignIn = vi.fn()
        render(<Live store={store} onSignIn={onSignIn}/>)

        await userEvent.click(screen.getByRole("button", {name: SEASON_EMAILS_CONSENT.words}))

        expect(onSignIn).toHaveBeenCalledTimes(1)
        expect(screen.queryByRole("textbox")).toBeNull()
    })

    it("shows a guest nothing where there is no sign-in pitch to open", async () => {
        const store = new SeasonEmailsStore(new FakeMarketingBackend())
        await store.follow({linked: []})

        const {container} = render(<Live store={store}/>)

        expect(container.innerHTML).toBe("")
    })

    it("fills the field with the account's address and subscribes it", async () => {
        await signedIn()

        expect((screen.getByLabelText("Email") as HTMLInputElement).value).toBe("ada@example.com")
        await userEvent.click(screen.getByRole("button", {name: SEASON_EMAILS_CONSENT.words}))

        expect(await screen.findByText("Emails on · ada@example.com")).toBeDefined()
        await userEvent.click(screen.getByRole("button", {name: "Turn off"}))
        expect(await screen.findByRole("button", {name: SEASON_EMAILS_CONSENT.words})).toBeDefined()
    })

    it("subscribes a typed address, which waits for its confirmation", async () => {
        const {store, backend} = await signedIn()

        const field = screen.getByLabelText("Email")
        await userEvent.clear(field)
        await userEvent.type(field, "ada@work.example")
        await userEvent.click(screen.getByRole("button", {name: SEASON_EMAILS_CONSENT.words}))

        expect(await screen.findByText("Confirm in the email sent to ada@work.example")).toBeDefined()

        backend.confirm()
        await act(() => store.refresh())
        expect(screen.getByText("Emails on · ada@work.example")).toBeDefined()
    })

    it("cancels a confirmation that waits", async () => {
        await signedIn()
        const field = screen.getByLabelText("Email")
        await userEvent.clear(field)
        await userEvent.type(field, "ada@work.example")
        await userEvent.click(screen.getByRole("button", {name: SEASON_EMAILS_CONSENT.words}))

        await userEvent.click(await screen.findByRole("button", {name: "Cancel"}))

        expect((await screen.findByLabelText("Email") as HTMLInputElement).value).toBe("ada@work.example")
    })

    it("keeps what was typed and says why the server refused it", async () => {
        await signedIn()
        const field = screen.getByLabelText("Email")
        await userEvent.clear(field)
        await userEvent.type(field, "not an address")

        await userEvent.click(screen.getByRole("button", {name: SEASON_EMAILS_CONSENT.words}))

        expect((await screen.findByRole("alert")).textContent).toBe("That is not an email address.")
        expect((screen.getByLabelText("Email") as HTMLInputElement).value).toBe("not an address")
    })

    it("cannot send an empty field", async () => {
        await signedIn(new FakeMarketingBackend())

        expect((screen.getByRole("button", {name: SEASON_EMAILS_CONSENT.words}) as HTMLButtonElement).disabled).toBe(true)
    })
})
