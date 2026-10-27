// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {Author} from "../gen/grpc/player/v1/internal_pb.ts"
import {SETTLE_MS} from "../home/ledger.ts"
import {OpenPlayer} from "../navigation.ts"
import {ANTIBOT_OFF} from "../players/actions.ts"
import {BattlePage} from "./BattlePage.tsx"

beforeEach(() => {
    vi.useFakeTimers({now: AT, toFake: ["Date"]})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

function show(admin: FakeAdmin) {
    const openPlayer = vi.fn()
    render(
        <OpenPlayer.Provider value={openPlayer}>
            <BattlePage clients={admin.clients()}/>
        </OpenPlayer.Provider>,
    )
    return {openPlayer}
}

function tile(label: string): HTMLElement {
    const found = screen.getByText(label, {selector: ".tile-label"}).closest("li")
    if (found === null) throw new Error(`no tile ${label}`)
    return found
}

function bans(): HTMLElement {
    return screen.getByRole("region", {name: "Bans"})
}

async function bansShown(): Promise<void> {
    await within(await screen.findByRole("region", {name: "Bans"})).findByText("BraveFox42")
}

function card(text: string): HTMLElement {
    const found = within(bans()).getAllByRole("listitem").find(item => item.classList.contains("card")
        && within(item).queryByText(text) !== null)
    if (found === undefined) throw new Error(`no ban of ${text}`)
    return found
}

async function reloaded(admin: FakeAdmin) {
    const asked = admin.askedFor("getMetrics").length
    fireEvent.click(screen.getByRole("button", {name: "Reload"}))
    await vi.waitFor(() => expect(admin.askedFor("getMetrics").length).toBeGreaterThan(asked))
    await screen.findByText("Updated", {exact: false})
}

// A request as the page sent it: its bounds are Timestamps.
type Sent = {since: {toDate(): Date}, until: {toDate(): Date}}

describe("the battle page", () => {
    it("lists the bans of the span, the latest first, each with what fired and who fell with it", async () => {
        const admin = new FakeAdmin()
        admin.authors = [new Author({accountId: ACCOUNT, name: "BraveFox42"}), new Author({accountId: "account-guest", name: "guest_a1b2c3"})]
        show(admin)

        const cards = await within(await screen.findByRole("region", {name: "Bans"})).findAllByRole("listitem")
        const shown = cards.filter(item => item.classList.contains("card"))
        expect(shown.map(item => within(item).getByText(/^(BraveFox42|guest_a1b2c3|2001:db8:1:2::\/64)$/).textContent))
            .toEqual(["2001:db8:1:2::/64", "guest_a1b2c3", "BraveFox42"])
        expect(within(card("BraveFox42")).getByText("catcher")).toBeTruthy()
        expect(within(card("BraveFox42")).getByText("catch median=400ms", {exact: false})).toBeTruthy()
        expect(within(card("BraveFox42")).queryByText("retaker"), "a watchdog that read clear is not a reason").toBeNull()
        expect(within(card("guest_a1b2c3")).getByText(/with BraveFox42/)).toBeTruthy()
        expect(within(tile("Bans")).getByText("3")).toBeTruthy()

        const asked = admin.askedFor("listConvictions") as Sent[]
        expect(asked[0].since.toDate()).toEqual(new Date(AT.getTime() - 2 * 3600_000))
    })

    it("opens a banned player on the player page, by its account or by its address", async () => {
        const admin = new FakeAdmin()
        const {openPlayer} = show(admin)
        await bansShown()

        fireEvent.click(within(card("BraveFox42")).getByRole("button", {name: "Open player"}))
        fireEvent.click(within(card("2001:db8:1:2::/64")).getByRole("button", {name: "Open player"}))

        expect(openPlayer.mock.calls).toEqual([
            [{caller: {accountId: ACCOUNT}, name: "BraveFox42"}],
            [{caller: {scope: "2001:db8:1:2::/64"}, name: undefined}],
        ])
    })

    it("lifts a ban only once the operator says yes, the account's and the address's apart", async () => {
        const admin = new FakeAdmin()
        show(admin)
        await bansShown()
        const banned = card("BraveFox42")

        fireEvent.click(within(banned).getByRole("button", {name: "Unban account"}))
        expect(admin.askedFor("unbanPlayer")).toEqual([])
        fireEvent.click(within(banned).getByRole("button", {name: "Yes, unban account"}))
        expect(await within(banned).findByText(/^Lifted offence 2/)).toBeTruthy()

        fireEvent.click(within(banned).getByRole("button", {name: "Unban address"}))
        fireEvent.click(within(banned).getByRole("button", {name: "Yes, unban address"}))
        await vi.waitFor(() => expect(admin.askedFor("unbanPlayer")).toHaveLength(2))

        expect(admin.askedFor("unbanPlayer")).toEqual([{accountId: ACCOUNT}, {scope: "203.0.113.7"}])
        expect(within(card("2001:db8:1:2::/64")).queryByRole("button", {name: "Unban account"}),
            "an address with no account has no account to lift").toBeNull()
    })

    it("charts the clicks a minute from the ledger and the refusals a minute from the counters, once two reads span one", async () => {
        const admin = new FakeAdmin()
        show(admin)

        expect(await screen.findByText(/^Last 2 hours:/)).toBeTruthy()
        expect(within(tile("Refused a minute")).getByText("…"), "one read counts nothing yet").toBeTruthy()
        expect(screen.getByRole("img", {name: "Clicks a minute: at most 1 a minute"})).toBeTruthy()

        admin.ledger.changes = []
        vi.setSystemTime(AT.getTime() + 60_000)
        await reloaded(admin)

        expect(await within(tile("Refused a minute")).findByText("33")).toBeTruthy()
        expect(within(tile("Refused a minute")).getByText("7.6% of clicks")).toBeTruthy()
        const legend = screen.getByText("Refused a minute", {selector: "figcaption"}).closest("figure")
        if (legend === null) throw new Error("no chart of refusals")
        expect(within(legend).getAllByRole("listitem").map(item => item.textContent)).toEqual(["throttled", "banned", "blocked"])
        expect(screen.getByText(/^Last 2 hours:/).textContent).toBe("Last 2 hours: 4 clicks")

        const ledger = admin.askedFor("getLedger") as Sent[]
        expect(ledger[1].since.toDate(), "the second read starts a minute before the first stopped")
            .toEqual(new Date(ledger[0].until.toDate().getTime() - SETTLE_MS))
    })

    it("reads the race, what the antibot reads now from its own counters, and the boxes from planet's", async () => {
        const admin = new FakeAdmin()
        show(admin)
        await screen.findByRole("region", {name: "Race"})
        await reloaded(admin)

        const race = screen.getByRole("region", {name: "Race"})
        expect(within(race).getByRole("heading", {name: "Round 12, the finale"})).toBeTruthy()
        expect(within(race).getByText("ends", {exact: false}).textContent).toContain("in 1 h 00 min")

        const antibot = screen.getByRole("region", {name: "Antibot"})
        const metronome = within(antibot).getByText("metronome").closest("tr")
        if (metronome === null) throw new Error("no metronome row")
        expect([...metronome.querySelectorAll("td")].map(cell => cell.textContent)).toEqual(["metronome", "4", "1", "0"])
        expect(within(antibot).getByText("2 bans running")).toBeTruthy()

        const catcher = within(antibot).getByText("catcher").closest("tr")
        if (catcher === null) throw new Error("no catcher row")
        expect(catcher.querySelectorAll("td")[3].textContent, "one ban since the page opened").toBe("1")

        const boxes = screen.getByRole("region", {name: "Boxes"})
        expect(within(boxes).getByText("25%")).toBeTruthy()
        expect(screen.queryByRole("region", {name: "Server"}), "no module owns the process").toBeNull()
    })

    it("shows who plays now", async () => {
        show(new FakeAdmin())

        expect(await screen.findByRole("region", {name: "Playing now"})).toBeTruthy()
        expect(within(tile("Playing now")).getByText("2")).toBeTruthy()
    })

    it("keeps every other read when one fails, and says which", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getRace", Code.Unavailable)
        show(admin)

        expect(await screen.findByText("The call failed.")).toBeTruthy()
        expect(await screen.findByRole("region", {name: "Bans"})).toBeTruthy()
        expect(screen.queryByRole("region", {name: "Race"})).toBeNull()
    })

    it("says the antibot is off rather than that it banned nobody", async () => {
        const admin = new FakeAdmin()
        admin.refuse("listConvictions", Code.Unimplemented)
        show(admin)

        expect(await screen.findByText(ANTIBOT_OFF)).toBeTruthy()
        expect(screen.queryByRole("region", {name: "Bans"})).toBeNull()
    })
})
