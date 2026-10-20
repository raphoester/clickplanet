// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {useState} from "react"
import {afterEach, describe, expect, it} from "vitest"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {InspectPlayerResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {CountryTiles} from "../gen/grpc/player/v1/player_pb.ts"
import {Opened} from "../navigation.ts"
import {PlayersPage} from "./PlayersPage.tsx"
import {NO_PROFILE} from "./ProfilePanel.tsx"

afterEach(cleanup)

function Page({admin, first}: {admin: FakeAdmin, first?: Opened}) {
    const [opened, setOpened] = useState(first)
    return <PlayersPage opened={opened} onOpen={setOpened} clients={admin.clients()}/>
}

function confirm(label: string, yes: string) {
    fireEvent.click(screen.getByRole("button", {name: label}))
    fireEvent.click(screen.getByRole("button", {name: yes}))
}

describe("the player list", () => {
    it("shows the top players first, each by its flag and its name rather than its account id", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin}/>)

        expect(await screen.findByText("2 of 75 players")).toBeTruthy()
        const [named, anonymous] = screen.getAllByRole("listitem")
        expect(within(named).getByText("BraveFox42")).toBeTruthy()
        expect(within(named).getByText("\u{1F1EB}\u{1F1F7}")).toBeTruthy()
        expect(within(named).getByText("France")).toBeTruthy()
        expect(screen.queryByText(ACCOUNT)).toBeNull()
        expect(within(anonymous).getByText("No account")).toBeTruthy()
        expect(within(anonymous).getByText("2001:db8:1:2::/64")).toBeTruthy()
        expect(admin.askedFor("topPlayers")).toEqual([{limit: 20}])
        expect(admin.askedFor("getAuthors")).toEqual([{accountIds: [ACCOUNT]}])
    })

    it("still lists the players when their names cannot be read, and says why", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getAuthors", Code.Unavailable)
        render(<Page admin={admin}/>)

        expect(await screen.findByText("The names could not be read. The call failed: refused")).toBeTruthy()
        expect(within(screen.getAllByRole("listitem")[0]).getByText("Unnamed")).toBeTruthy()
    })

    it("finds who painted a flag on a country's ground", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin}/>)
        await screen.findByText("2 of 75 players")

        fireEvent.click(screen.getByRole("button", {name: "By flag"}))
        fireEvent.change(screen.getByRole("textbox", {name: "Flag"}), {target: {value: "PS"}})
        fireEvent.change(screen.getByRole("textbox", {name: "On the ground of"}), {target: {value: "il"}})
        expect(screen.getByText("Palestinian Territories")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Find"}))

        expect(await screen.findByText("2 players")).toBeTruthy()
        expect(admin.askedFor("findPlayers")).toEqual([{flagCountryId: "ps", areaCountryId: "il", limit: 20}])
    })

    it("opens a row by its account, and its address from there", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin}/>)
        await screen.findByText("2 of 75 players")

        fireEvent.click(within(screen.getAllByRole("listitem")[0]).getByRole("button", {name: "Open"}))
        expect(await screen.findByText(/^0 of 0$/)).toBeTruthy()
        expect(screen.getByRole("heading", {name: "BraveFox42 France"})).toBeTruthy()
        expect(admin.askedFor("inspectPlayer")).toEqual([{accountId: ACCOUNT}])

        const card = screen.getByRole("article")
        fireEvent.click(within(card).getByRole("button", {name: "Open"}))
        await screen.findByRole("heading", {name: "Ban"})
        expect(admin.askedFor("inspectPlayer")).toEqual([{accountId: ACCOUNT}, {scope: "203.0.113.7"}])
    })

    it("opens what the operator typed: an account id or an address", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin}/>)

        fireEvent.change(screen.getByRole("textbox", {name: "Account id or address"}), {target: {value: " 198.51.100.4 "}})
        fireEvent.click(screen.getByRole("button", {name: "Open"}))

        await screen.findByRole("heading", {name: "Ban"})
        expect(admin.askedFor("inspectPlayer")).toEqual([{scope: "198.51.100.4"}])
    })
})

describe("one player", () => {
    const account: Opened = {caller: {accountId: ACCOUNT}, name: "BraveFox42"}
    const address: Opened = {caller: {scope: "203.0.113.7"}}

    it("reads what the game keeps about the account, under the name it holds now", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={{caller: {accountId: ACCOUNT}, name: "OldName"}}/>)

        const profile = await screen.findByRole("region", {name: "Profile"})
        expect(await within(profile).findByText(/^Signed in, made/)).toBeTruthy()
        expect(within(profile).getByText("(username)")).toBeTruthy()
        expect(within(profile).getByText("1200")).toBeTruthy()
        expect(within(profile).getByText("3 days now, best 12 days")).toBeTruthy()
        expect(within(profile).getByText("France")).toBeTruthy()
        expect(within(profile).getByText("Scotland")).toBeTruthy()
        expect(screen.getByRole("heading", {name: "BraveFox42 France"})).toBeTruthy()
        expect(admin.askedFor("getPlayer")).toEqual([{accountId: ACCOUNT}])
    })

    it("folds the countries past five behind All", async () => {
        const admin = new FakeAdmin()
        admin.profile.playsAgainst = ["de", "es", "it", "pt", "be", "nl", "ch"]
            .map((countryId, i) => new CountryTiles({countryId, tiles: BigInt(70 - i)}))
        render(<Page admin={admin} first={account}/>)

        const profile = await screen.findByRole("region", {name: "Profile"})
        await within(profile).findByText("Germany")
        expect(within(profile).queryByText("Netherlands")).toBeNull()

        fireEvent.click(within(profile).getByRole("button", {name: "All 7"}))
        expect(within(profile).getByText("Switzerland")).toBeTruthy()
    })

    it("says when the game keeps nothing about an account, and reads no profile for an address", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getPlayer", Code.NotFound)
        render(<Page admin={admin} first={account}/>)
        expect(await screen.findByText(NO_PROFILE)).toBeTruthy()
        cleanup()

        render(<Page admin={admin} first={address}/>)
        await screen.findByRole("heading", {name: "Ban"})
        expect(screen.queryByRole("region", {name: "Profile"})).toBeNull()
        expect(admin.askedFor("getPlayer")).toHaveLength(1)
    })

    it("says the antibot is off rather than failing", async () => {
        const admin = new FakeAdmin()
        admin.refuse("inspectPlayer", Code.FailedPrecondition)
        render(<Page admin={admin} first={account}/>)

        expect(await screen.findByText("The antibot is off on this server, so it cannot ban, unban or inspect.")).toBeTruthy()
    })

    it("bans, then reads the antibot again", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)
        await screen.findByText(/^42 in /)

        confirm("Ban", "Yes, ban")

        expect(await screen.findByText(/^Banned until .*, offence 2\.$/)).toBeTruthy()
        expect(admin.askedFor("banPlayer")).toEqual([{accountId: ACCOUNT, duration: undefined}])
        expect(admin.askedFor("inspectPlayer")).toHaveLength(2)
    })

    it("shows a pardon only while one runs", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)
        await screen.findByText(/^42 in /)
        expect(screen.queryByText("Pardoned")).toBeNull()
        cleanup()

        admin.inspection = new InspectPlayerResponse({clicks: 42, banned: false, pardonedUntil: Timestamp.fromDate(AT)})
        render(<Page admin={admin} first={account}/>)
        expect(await screen.findByText("Pardoned")).toBeTruthy()
    })

    it("lifts the latest ban, and says when there is none", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={address}/>)

        confirm("Unban", "Yes, unban")
        expect(await screen.findByText(/^Lifted offence 2: its ban was until .*\. Pardoned until .*\.$/)).toBeTruthy()
        expect(admin.askedFor("unbanPlayer")).toEqual([{scope: "203.0.113.7"}])

        admin.refuse("unbanPlayer", Code.NotFound)
        confirm("Unban", "Yes, unban")
        expect(await screen.findByText("This player has no ban left to lift.")).toBeTruthy()
    })

    it("reverts only after a dry run, and asks with its numbers", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)

        const revert = () => screen.getByRole("button", {name: "Revert"}) as HTMLButtonElement
        expect(revert().disabled).toBe(true)

        fireEvent.click(screen.getByRole("button", {name: "Dry run"}))
        expect(await screen.findByText(/took 900 tiles in the last 72 hours and still holds 120/)).toBeTruthy()

        fireEvent.click(revert())
        expect(screen.getByText("Give back the 120 tiles BraveFox42 holds?")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, revert"}))

        expect(await screen.findByText(/^Restored 118 of 120 tiles held\. The rest were retaken/)).toBeTruthy()
        expect(admin.askedFor("revertPlayer")).toEqual([
            {accountId: ACCOUNT, dryRun: true},
            {accountId: ACCOUNT, dryRun: false},
        ])
    })

    it("grants the charges picked, and says what fit", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)

        const grant = () => screen.getByRole("button", {name: "Grant"}) as HTMLButtonElement
        expect(grant().disabled).toBe(true)

        fireEvent.click(screen.getByRole("checkbox", {name: "Bomb"}))
        fireEvent.change(screen.getByRole("spinbutton", {name: "Enclosures"}), {target: {value: "3"}})
        fireEvent.click(grant())
        expect(screen.getByText("Give BraveFox42 a bomb and 3 enclosures?")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, grant"}))

        expect(await screen.findByText("It held 2 enclosures. It holds a bomb and 3 enclosures.")).toBeTruthy()
        expect(admin.askedFor("grantCharges")).toEqual([{
            accountId: ACCOUNT, refill: false, bomb: true, enclosures: 3, spreadClicks: 0, shields: 0,
        }])
    })

    it("mutes an account, and offers neither a mute nor charges to an address", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)
        confirm("Mute", "Yes, mute")
        expect(await screen.findByText(/^Muted until /)).toBeTruthy()
        cleanup()

        render(<Page admin={admin} first={address}/>)
        expect(screen.queryByRole("button", {name: "Mute"})).toBeNull()
        expect(screen.queryByRole("button", {name: "Grant"})).toBeNull()
    })
})
