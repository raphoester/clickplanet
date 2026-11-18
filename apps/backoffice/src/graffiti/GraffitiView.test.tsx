// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {useState} from "react"
import {afterEach, describe, expect, it} from "vitest"
import {Places} from "../boards/places.ts"
import {FIXTURE_1} from "../boards/testBoards.ts"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {InspectPlayerResponse} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {Graffiti, GraffitiKind} from "../gen/grpc/vandalsquad/v1/admin_pb.ts"
import {keyOf} from "./graffiti.ts"
import {GraffitiView} from "./GraffitiView.tsx"
import {patch} from "./testPlaces.ts"

afterEach(cleanup)

const {places, idOf} = patch(51.6, 20.1)
const PLUS = [idOf(0, 0), idOf(1, 0), idOf(-1, 0), idOf(0, 1), idOf(0, -1)]
// A patch of fixture-1, the tests' coarse board: 19 tiles.
const fixture = patch(6.5, 3.4, 2, FIXTURE_1)

const graffitiOf = (accountId: string, tiles: number[], minutesAgo: number, board = "earth-1") => new Graffiti({
    board, accountId, countryId: "de", kind: GraffitiKind.SWASTIKA, tileIds: tiles,
    flaggedAt: Timestamp.fromDate(new Date(AT.getTime() - minutesAgo * 60_000)),
    seenAt: Timestamp.fromDate(AT), link: "https://clickplanet.lol/play?lat=51.60&lon=20.11&zoom=20.0",
})

// Two of BraveFox42's, newest first, then one of an account the game never named.
function withGraffiti(): FakeAdmin {
    const admin = new FakeAdmin()
    admin.graffiti = [
        graffitiOf(ACCOUNT, PLUS, 5),
        graffitiOf(ACCOUNT, [idOf(3, 3), idOf(4, 3), idOf(3, 4)], 60),
        graffitiOf("account-unnamed", [idOf(-4, -4), idOf(-5, -4)], 120),
    ]
    return admin
}

// Each board's patch by its name, and the names asked.
function boardsRead() {
    const asked: string[] = []
    const read = (board: string): Promise<Places> => {
        asked.push(board)
        if (board === "earth-1") return Promise.resolve(places)
        if (board === "fixture-1") return Promise.resolve(fixture.places)
        return Promise.reject(new Error(`${board} is not a board this page carries`))
    }
    return {asked, read}
}

function Page({admin, first = "", read = boardsRead().read}: {admin: FakeAdmin, first?: string, read?: (board: string) => Promise<Places>}) {
    const [opened, setOpened] = useState(first)
    return (
        <>
            <p data-testid="opened">{opened}</p>
            <GraffitiView opened={opened} onOpen={setOpened} clients={admin.clients()} places={read}/>
        </>
    )
}

const drawnIn = (picture: HTMLElement) => picture.querySelectorAll("polygon[data-drawn]").length

describe("the graffiti page", () => {
    it("lists each graffiti drawn small, with its painter, how many it painted, and whether it is banned", async () => {
        const admin = withGraffiti()
        render(<Page admin={admin}/>)

        const list = within(await screen.findByRole("list", {name: "The graffiti"}))
        const rows = list.getAllByRole("listitem")
        expect(rows).toHaveLength(3)
        expect(rows[0].textContent).toContain("BraveFox42")
        expect(within(rows[0]).getByText("×2")).toBeTruthy()
        expect(rows[0].textContent).toContain("5 tiles")
        expect(await within(rows[0]).findByText(/not banned/)).toBeTruthy()
        expect(drawnIn(await within(rows[0]).findByRole("img"))).toBe(5)
        expect(rows[2].textContent).toContain("Unnamed")
        expect(admin.askedFor("getPlayers")).toEqual([{accountIds: [ACCOUNT, "account-unnamed"]}])
        expect(admin.askedFor("listGraffiti")).toHaveLength(1)
        expect(screen.getByText("Pick a graffiti on the left.")).toBeTruthy()
    })

    it("opens one picked: the drawing over the land around it, who painted it, and where it stands", async () => {
        const admin = withGraffiti()
        render(<Page admin={admin}/>)

        fireEvent.click((await screen.findAllByRole("button", {name: /BraveFox42/}))[0])
        expect(screen.getByTestId("opened").textContent).toBe(keyOf(admin.graffiti[0]))
        const detail = within(screen.getByRole("region", {name: "Graffiti"}))
        const picture = await detail.findByRole("img", {name: /The 5 tiles BraveFox42 painted/})
        expect(drawnIn(picture)).toBe(5)
        expect(picture.querySelectorAll("polygon").length).toBeGreaterThan(40)
        expect(detail.getByRole("link", {name: "Open on the globe"}).getAttribute("href")).toBe(admin.graffiti[0].link)
        expect(detail.getByText("Germany")).toBeTruthy()
        expect(await detail.findByText("Not banned")).toBeTruthy()
        expect(detail.getByText("One more by this painter:")).toBeTruthy()
    })

    it("opens the painter's other graffiti from the one open", async () => {
        const admin = withGraffiti()
        render(<Page admin={admin} first={keyOf(admin.graffiti[0])}/>)

        const detail = within(await screen.findByRole("region", {name: "Graffiti"}))
        fireEvent.click(detail.getByRole("button", {name: /3 tiles/}))
        expect(screen.getByTestId("opened").textContent).toBe(keyOf(admin.graffiti[1]))
    })

    it("bans the painter after asking, then reads where it stands again", async () => {
        const admin = withGraffiti()
        render(<Page admin={admin} first={keyOf(admin.graffiti[0])}/>)

        const tools = within(await screen.findByRole("region", {name: "Tools"}))
        fireEvent.click(tools.getByRole("button", {name: "Ban"}))
        expect(admin.askedFor("banPlayer")).toHaveLength(0)
        expect(tools.getByText("Ban BraveFox42 for the ladder's next step?")).toBeTruthy()
        const inspected = admin.askedFor("inspectPlayer").length
        fireEvent.click(tools.getByRole("button", {name: "Yes, ban"}))

        expect(await tools.findByText(/Banned until .*, offence 2\./)).toBeTruthy()
        expect(admin.askedFor("banPlayer")).toEqual([{accountId: ACCOUNT, duration: undefined}])
        await waitFor(() => expect(admin.askedFor("inspectPlayer").length).toBeGreaterThan(inspected))
    })

    it("reverts the painter only after a dry run, with its numbers", async () => {
        const admin = withGraffiti()
        render(<Page admin={admin} first={keyOf(admin.graffiti[0])}/>)

        const tools = within(await screen.findByRole("region", {name: "Tools"}))
        fireEvent.click(tools.getByRole("tab", {name: "Revert"}))
        expect((tools.getByRole("button", {name: "Revert"}) as HTMLButtonElement).disabled).toBe(true)
        fireEvent.click(tools.getByRole("button", {name: "Dry run"}))
        expect(await tools.findByText(/still holds 120/)).toBeTruthy()
        fireEvent.click(tools.getByRole("button", {name: "Revert"}))
        expect(tools.getByText("Give back the 120 tiles BraveFox42 holds?")).toBeTruthy()
        fireEvent.click(tools.getByRole("button", {name: "Yes, revert"}))

        expect(await tools.findByText(/^Restored/)).toBeTruthy()
        expect(admin.askedFor("revertPlayer")).toEqual([
            {accountId: ACCOUNT, dryRun: true},
            {accountId: ACCOUNT, dryRun: false},
        ])
    })

    it("keeps to the painters not banned yet, on a click", async () => {
        const admin = withGraffiti()
        admin.inspection = new InspectPlayerResponse({accountId: ACCOUNT, banned: true, bannedUntil: Timestamp.fromDate(AT), offence: 1})
        render(<Page admin={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Painter not banned (0)"}))
        expect(screen.getByText("Every painter is banned.")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "All (3)"}))
        expect(within(screen.getByRole("list", {name: "The graffiti"})).getAllByText("banned")).toHaveLength(3)
    })

    it("says the squad is off, and a graffiti no longer kept", async () => {
        const off = withGraffiti()
        off.refuse("listGraffiti", Code.Unimplemented)
        render(<Page admin={off}/>)
        expect(await screen.findByText(/The vandal squad is off on this server/)).toBeTruthy()
        cleanup()

        render(<Page admin={withGraffiti()} first="0-de-gone"/>)
        expect(await screen.findByText("This graffiti is not kept any more.")).toBeTruthy()
    })

    it("draws each graffiti on the board it names, each board read once", async () => {
        const admin = new FakeAdmin()
        admin.graffiti = [
            graffitiOf(ACCOUNT, PLUS, 5),
            graffitiOf(ACCOUNT, [fixture.idOf(0, 0), fixture.idOf(1, 0), fixture.idOf(0, 1)], 10, "fixture-1"),
            // An id of earth-1's patch, past fixture-1's 19.
            graffitiOf("account-unnamed", [idOf(5, 5)], 20, "fixture-1"),
        ]
        const {asked, read} = boardsRead()
        render(<Page admin={admin} read={read}/>)

        const list = within(await screen.findByRole("list", {name: "The graffiti"}))
        const rows = list.getAllByRole("listitem")
        expect(drawnIn(await within(rows[0]).findByRole("img"))).toBe(5)
        expect(drawnIn(await within(rows[1]).findByRole("img"))).toBe(3)
        expect(within(rows[2]).getByText("None of its tiles is on fixture-1.")).toBeTruthy()
        expect(asked).toEqual(["earth-1", "fixture-1"])

        fireEvent.click(rows[1].querySelector("button")!)
        const detail = within(screen.getByRole("region", {name: "Graffiti"}))
        expect(detail.getByText("fixture-1")).toBeTruthy()
        // Its land is fixture-1's patch, not earth-1's.
        expect((await detail.findByRole("img", {name: /The 3 tiles BraveFox42 painted/})).querySelectorAll("polygon").length)
            .toBeLessThanOrEqual(19)
    })

    it("says so of a board the page does not carry, and draws the rest", async () => {
        const admin = withGraffiti()
        admin.graffiti[1] = graffitiOf(ACCOUNT, [idOf(3, 3)], 60, "earth-9")
        render(<Page admin={admin} first={keyOf(admin.graffiti[1])}/>)

        expect((await screen.findAllByText("earth-9 is not a board this page carries")).length).toBeGreaterThan(0)
        const detail = within(screen.getByRole("region", {name: "Graffiti"}))
        expect(detail.getByText(/^Its tiles could not be placed: earth-9 is not a board/)).toBeTruthy()
        const rows = within(screen.getByRole("list", {name: "The graffiti"})).getAllByRole("listitem")
        expect(drawnIn(await within(rows[0]).findByRole("img"))).toBe(5)
    })
})
