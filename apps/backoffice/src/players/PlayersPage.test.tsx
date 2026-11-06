// @vitest-environment jsdom
import {textBody} from "../chat/said.ts"
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {useState} from "react"
import {afterEach, describe, expect, it} from "vitest"
import {AROUND} from "../chat/theirConversations.ts"
import {timeOf} from "../components/time.ts"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {AdminGetPlayerResponse} from "../gen/grpc/player/v1/admin_pb.ts"
import {GetMuteResponse, KeptMessage} from "../gen/grpc/chat/v1/admin_pb.ts"
import {MuteKind} from "../gen/grpc/chat/v1/chat_pb.ts"
import {InspectPlayerResponse} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {Player as Identity} from "../gen/grpc/player/v1/player_pb.ts"
import {AdminGetProgressResponse} from "../gen/grpc/progression/v1/admin_pb.ts"
import {CountryTiles, Stats} from "../gen/grpc/progression/v1/progression_pb.ts"
import {Opened} from "../navigation.ts"
import {ALERTS_OFF, NO_USERNAME} from "./actions.ts"
import {PlayersPage} from "./PlayersPage.tsx"
import {NO_PROFILE} from "./ProfilePanel.tsx"

afterEach(cleanup)

function Page({admin, first}: {admin: FakeAdmin, first?: Opened}) {
    const [opened, setOpened] = useState(first)
    return <PlayersPage opened={opened} onOpen={setOpened} clients={admin.clients()}/>
}

function said(n: number, accountId = ACCOUNT, authorName = "BraveFox42") {
    return new KeptMessage({
        messageId: `${accountId}-${n}`, accountId, authorName, body: textBody(`${authorName} ${n}`),
        sentAt: Timestamp.fromDate(new Date(AT.getTime() + n * 60_000)), countryId: "fr",
    })
}

function tool(name: string) {
    fireEvent.click(screen.getByRole("tab", {name}))
}

function confirm(label: string, yes: string) {
    fireEvent.click(screen.getByRole("button", {name: label}))
    fireEvent.click(screen.getByRole("button", {name: yes}))
}

// The rows of the list, its head left out.
function rows(): HTMLElement[] {
    return screen.getAllByRole("row").slice(1)
}

function search(typed: string) {
    fireEvent.change(screen.getByRole("searchbox", {name: "Name, account id or address"}), {target: {value: typed}})
}

describe("the player list", () => {
    it("shows the top players first, each by its flag and its name rather than its account id", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin}/>)

        expect(await screen.findByText("2 of 75 players")).toBeTruthy()
        const [named, anonymous] = rows()
        expect(within(named).getByText("BraveFox42")).toBeTruthy()
        expect(within(named).getByText("\u{1F1EB}\u{1F1F7}")).toBeTruthy()
        expect(within(named).getByText("France")).toBeTruthy()
        expect(screen.queryByText(ACCOUNT)).toBeNull()
        expect(within(anonymous).getByText("No account")).toBeTruthy()
        expect(within(anonymous).getByText("2001:db8:1:2::/64")).toBeTruthy()
        expect(admin.askedFor("topPlayers")).toEqual([{limit: 20}])
        expect(admin.askedFor("getPlayers")).toEqual([{accountIds: [ACCOUNT]}])
    })

    it("sorts the rows read by a column, most first, then least first", async () => {
        render(<Page admin={new FakeAdmin()}/>)

        await screen.findByText("2 of 75 players")
        fireEvent.click(screen.getByRole("button", {name: "Held"}))
        expect(within(rows()[0]).getByText("No account")).toBeTruthy()
        expect(screen.getByRole("columnheader", {name: "Held ▾"}).getAttribute("aria-sort")).toBe("descending")
        fireEvent.click(screen.getByRole("button", {name: "Held ▾"}))
        expect(within(rows()[0]).getByText("BraveFox42")).toBeTruthy()
    })

    it("still lists the players when their names cannot be read, and says why", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getPlayers", Code.Unavailable)
        render(<Page admin={admin}/>)

        expect(await screen.findByText("The names could not be read. The call failed: refused")).toBeTruthy()
        expect(within(rows()[0]).getByText("Unnamed")).toBeTruthy()
    })

    it("finds who painted a flag on a country's ground, and goes back to the most active", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin}/>)

        await screen.findByText("2 of 75 players")
        fireEvent.change(screen.getByRole("textbox", {name: "Flag"}), {target: {value: "PS"}})
        fireEvent.change(screen.getByRole("textbox", {name: "On the ground of"}), {target: {value: "il"}})
        expect(screen.getByText("Palestinian Territories")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Filter"}))

        expect(await screen.findByText("2 players")).toBeTruthy()
        expect(screen.getByRole("heading", {name: "Took tiles for Palestinian Territories on Israel's ground"})).toBeTruthy()
        expect(admin.askedFor("findPlayers")).toEqual([{flagCountryId: "ps", areaCountryId: "il", limit: 20}])
        fireEvent.click(screen.getByRole("button", {name: "× Back to the most active"}))
        expect(await screen.findByText("2 of 75 players")).toBeTruthy()
    })

    it("finds any account by a part of its name, and opens it", async () => {
        const admin = new FakeAdmin()
        admin.identities.push(new Identity({accountId: "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60", name: "guest_a1b2c3", guest: true}))
        render(<Page admin={admin}/>)

        await screen.findByText("2 of 75 players")
        expect((screen.getByRole("button", {name: "Find"}) as HTMLButtonElement).disabled).toBe(true)
        search(" GUEST_A1 ")
        fireEvent.click(screen.getByRole("button", {name: "Find"}))
        expect(await screen.findByText("1 account")).toBeTruthy()
        expect(within(rows()[0]).getByText("guest code")).toBeTruthy()
        expect(admin.askedFor("searchPlayers")).toEqual([{name: "GUEST_A1", limit: 20}])

        search("brave")
        fireEvent.click(screen.getByRole("button", {name: "Find"}))
        const row = within(await screen.findByRole("row", {name: /BraveFox42/}))
        expect(row.getByText("username")).toBeTruthy()
        fireEvent.click(row.getByRole("button", {name: "BraveFox42"}))
        expect(await screen.findByRole("heading", {name: "BraveFox42 France"})).toBeTruthy()
        expect(admin.askedFor("inspectPlayer")).toEqual([{accountId: ACCOUNT}])
    })

    it("says when no name holds what was typed", async () => {
        render(<Page admin={new FakeAdmin()}/>)

        search("nobody")
        fireEvent.click(screen.getByRole("button", {name: "Find"}))
        expect(await screen.findByText("No account has a name holding “nobody”.")).toBeTruthy()
    })

    it("says when a name search fails, and tries again", async () => {
        const admin = new FakeAdmin()
        admin.refuse("searchPlayers", Code.Unavailable)
        render(<Page admin={admin}/>)

        await screen.findByText("2 of 75 players")
        search("brave")
        fireEvent.click(screen.getByRole("button", {name: "Find"}))
        fireEvent.click(await screen.findByRole("button", {name: "Try again"}))
        expect(admin.askedFor("searchPlayers")).toHaveLength(2)
    })

    it("opens a row by its account, and its address from there", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin}/>)

        await screen.findByText("2 of 75 players")
        fireEvent.click(within(rows()[0]).getByRole("button", {name: "BraveFox42 France"}))
        expect(await screen.findByText(/^0 of 0$/)).toBeTruthy()
        expect(screen.getByRole("heading", {name: "BraveFox42 France"})).toBeTruthy()
        expect(admin.askedFor("inspectPlayer")).toEqual([{accountId: ACCOUNT}])

        const card = screen.getByRole("article")
        fireEvent.click(within(card).getByRole("button", {name: "Open"}))
        expect(await screen.findByRole("heading", {name: "203.0.113.7"})).toBeTruthy()
        await screen.findByText(/^0 of 0$/)
        expect(admin.askedFor("inspectPlayer")).toEqual([{accountId: ACCOUNT}, {scope: "203.0.113.7"}])
    })

    it("opens what the operator typed: an account id or an address", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin}/>)

        search(" 198.51.100.4 ")
        fireEvent.click(screen.getByRole("button", {name: "Open"}))
        await screen.findByText(/^0 of 0$/)
        expect(admin.askedFor("inspectPlayer")).toEqual([{scope: "198.51.100.4"}])
        expect(admin.askedFor("searchPlayers")).toEqual([])
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
        expect(admin.askedFor("getProgress")).toEqual([{accountId: ACCOUNT}])
    })

    it("asks who the account is and how far it got at once", async () => {
        const admin = new FakeAdmin()
        const release = admin.hold("getPlayer")
        render(<Page admin={admin} first={account}/>)

        expect(admin.askedFor("getProgress")).toEqual([{accountId: ACCOUNT}])
        release()
        const profile = await screen.findByRole("region", {name: "Profile"})
        expect(await within(profile).findByText("1200")).toBeTruthy()
    })

    it("folds the countries past five behind All", async () => {
        const admin = new FakeAdmin()
        admin.progress.playsAgainst = ["de", "es", "it", "pt", "be", "nl", "ch"]
            .map((countryId, i) => new CountryTiles({countryId, tiles: BigInt(70 - i)}))
        render(<Page admin={admin} first={account}/>)

        const profile = await screen.findByRole("region", {name: "Profile"})
        await within(profile).findByText("Germany")
        expect(within(profile).queryByText("Netherlands")).toBeNull()

        fireEvent.click(within(profile).getByRole("button", {name: "All 7"}))
        expect(within(profile).getByText("Switzerland")).toBeTruthy()
    })

    it("shows the progress of an account with no name and no guest code", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getPlayer", Code.NotFound)
        render(<Page admin={admin} first={{caller: {accountId: ACCOUNT}}}/>)

        const profile = await screen.findByRole("region", {name: "Profile"})
        expect(await within(profile).findByText("none: it never joined the roster or posted, so it has no guest code")).toBeTruthy()
        expect(within(profile).getByText("1200")).toBeTruthy()
        expect(within(profile).queryByText("Account")).toBeNull()
        expect(screen.getByRole("heading", {name: "Unnamed France"})).toBeTruthy()
    })

    it("names none an account that never got a name or a guest code, and says what auth knows of it", async () => {
        const admin = new FakeAdmin()
        admin.profile = new AdminGetPlayerResponse({guest: true, createdAtUnixMs: BigInt(AT.getTime())})
        render(<Page admin={admin} first={{caller: {accountId: ACCOUNT}}}/>)

        const profile = await screen.findByRole("region", {name: "Profile"})
        expect(await within(profile).findByText("none: it never joined the roster or posted, so it has no guest code")).toBeTruthy()
        expect(within(profile).getByText("Account")).toBeTruthy()
        expect(within(profile).getByText("1200")).toBeTruthy()
    })

    it("says when the game keeps nothing about an account, and reads no profile for an address", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getPlayer", Code.NotFound)
        admin.progress = new AdminGetProgressResponse({stats: new Stats()})
        render(<Page admin={admin} first={account}/>)
        expect(await screen.findByText(NO_PROFILE)).toBeTruthy()
        cleanup()

        render(<Page admin={admin} first={address}/>)
        await screen.findByText(/^0 of 0$/)
        expect(screen.queryByRole("region", {name: "Profile"})).toBeNull()
        expect(admin.askedFor("getPlayer")).toHaveLength(1)
        expect(admin.askedFor("getProgress")).toHaveLength(1)
    })

    it("shows what the account said in its conversations on asking, read by its account id, no one tagged", async () => {
        const admin = new FakeAdmin()
        admin.chat = [said(1, "account-cleo", "Cleo"), said(2), said(3, "account-dan", "Dan")]
        render(<Page admin={admin} first={account}/>)

        const messages = within(await screen.findByRole("region", {name: "Their messages"}))
        const fold = await messages.findByRole("button", {name: /^What BraveFox42 said/})
        expect(await messages.findByText("4 messages")).toBeTruthy()
        expect(admin.askedFor("listMessages")).toEqual([])
        fireEvent.click(fold)
        const list = await messages.findByRole("list", {name: "Their conversations, newest first"})
        const shown = within(list).getAllByRole("list", {name: "A conversation"}).map(conversation =>
            within(conversation).getAllByText(/^\w+ \d+$/, {selector: ".line-text"}).map(line => line.textContent))
        expect(shown).toEqual([["Cleo 1", "BraveFox42 2", "Dan 3"]])
        expect(messages.getByText("BraveFox42 2").closest("li")?.className).toContain("line-theirs")
        expect(messages.queryByText("reported")).toBeNull()
        expect(messages.queryByText("reporter")).toBeNull()
        expect(admin.askedFor("listMessages")).toEqual([
            {accountId: ACCOUNT, limit: 20},
            {beforeMessageId: `${ACCOUNT}-2`, limit: AROUND},
            {afterMessageId: `${ACCOUNT}-2`, limit: AROUND},
        ])
    })

    it("reads no messages for an address: the chat keeps them by account", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={address}/>)

        await screen.findByText(/^0 of 0$/)
        expect(screen.queryByRole("region", {name: "Their messages"})).toBeNull()
        expect(admin.askedFor("listMessages")).toEqual([])
    })

    it("says the antibot is off rather than failing", async () => {
        const admin = new FakeAdmin()
        admin.refuse("inspectPlayer", Code.Unimplemented)
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

    it("says at the top whether the player is banned, and shows a pardon only while one runs", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)
        await screen.findByText(/^42 in /)
        expect(screen.getByText("Not banned")).toBeTruthy()
        expect(screen.queryByText("Pardoned")).toBeNull()
        cleanup()

        admin.inspection = new InspectPlayerResponse({clicks: 42, banned: false, pardonedUntil: Timestamp.fromDate(AT)})
        render(<Page admin={admin} first={account}/>)
        expect(await screen.findByText("Pardoned")).toBeTruthy()
    })

    it("lifts the latest ban, and says when there is none", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={address}/>)

        tool("Unban")
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

        tool("Revert")
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

        tool("Charges")
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

    it("maxes out every charge in one tap, and says what fit", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)

        tool("Charges")
        fireEvent.click(screen.getByRole("button", {name: "Max out"}))
        expect(screen.getByText("Give BraveFox42 every charge, as many as fit?")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, max out"}))

        expect(await screen.findByText("It held 2 enclosures. It holds a bomb and 3 enclosures.")).toBeTruthy()
        expect(admin.askedFor("grantCharges")).toEqual([{
            accountId: ACCOUNT, refill: true, bomb: true, enclosures: 99, spreadClicks: 99, shields: 99,
        }])
    })

    it("lets an account change flag at once, and says when the rules lock every flag", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)

        tool("Flag")
        fireEvent.click(screen.getByRole("button", {name: "Allow a change now"}))
        expect(screen.getByText("Let BraveFox42 change flag now, without the hour of wait?")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, allow a change now"}))
        expect(await screen.findByText("Done. The next change of flag goes at once.")).toBeTruthy()
        expect(admin.askedFor("releaseAllegiance")).toEqual([{accountId: ACCOUNT}])
        cleanup()

        admin.flagsLocked = true
        render(<Page admin={admin} first={account}/>)
        tool("Flag")
        confirm("Allow a change now", "Yes, allow a change now")
        expect(await screen.findByText("Done, but the rules in force lock every flag: the change waits for them to end.")).toBeTruthy()
    })

    it("renames an account and shows the new name at the top", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)

        tool("Name")
        expect((screen.getByRole("button", {name: "Rename"}) as HTMLButtonElement).disabled).toBe(true)
        fireEvent.change(screen.getByRole("textbox", {name: "New name"}), {target: {value: " Ada_L "}})
        fireEvent.click(screen.getByRole("button", {name: "Rename"}))
        expect(screen.getByText("Rename BraveFox42 to Ada_L?")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, rename"}))

        expect(await screen.findByText("Renamed to Ada_L.")).toBeTruthy()
        expect(admin.askedFor("renamePlayer")).toEqual([{accountId: ACCOUNT, name: "Ada_L", reason: ""}])
        expect(await screen.findByRole("heading", {name: "Ada_L France"})).toBeTruthy()
    })

    it("says when the account has no username to change", async () => {
        const admin = new FakeAdmin()
        admin.refuse("renamePlayer", Code.NotFound)
        render(<Page admin={admin} first={account}/>)

        tool("Name")
        fireEvent.change(screen.getByRole("textbox", {name: "New name"}), {target: {value: "Ada"}})
        confirm("Rename", "Yes, rename")

        expect(await screen.findByText(NO_USERNAME)).toBeTruthy()
    })

    it("shows an account's Discord link, unlinks it, and says it is not linked any more", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)

        tool("Discord")
        expect(await screen.findByText("80351110224678912")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Unlink Discord"}))
        expect(screen.getByText(/^Unlink BraveFox42 from Discord\?/)).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, unlink discord"}))

        expect(await screen.findByText(/Not linked: the game offers BraveFox42 to join Discord/)).toBeTruthy()
        expect(admin.askedFor("unlinkPlayer")).toEqual([{accountId: ACCOUNT}])
        expect(admin.askedFor("getPlayerLink")).toEqual([{accountId: ACCOUNT}, {accountId: ACCOUNT}])
    })

    it("says Discord pings are off when the alerts module is", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getPlayerLink", Code.Unimplemented)
        render(<Page admin={admin} first={account}/>)

        tool("Discord")

        expect(await screen.findByText(ALERTS_OFF)).toBeTruthy()
    })

    it("mutes an account, and offers neither a mute nor charges to an address", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)
        tool("Mute")
        confirm("Mute", "Yes, mute")
        expect(await screen.findByText(/^Muted until /)).toBeTruthy()
        cleanup()

        render(<Page admin={admin} first={address}/>)
        expect(screen.getAllByRole("tab").map(tab => tab.textContent)).toEqual(["Ban", "Unban", "Revert"])
        expect(screen.queryByText("Not muted")).toBeNull()
    })

    it("says until when an account is muted, unmutes it, and reads it again", async () => {
        const admin = new FakeAdmin()
        admin.muting = new GetMuteResponse({mutedUntil: Timestamp.fromDate(AT), mutedAccountId: ACCOUNT})
        render(<Page admin={admin} first={account}/>)
        expect(await screen.findByText(`Muted until ${timeOf(AT)}`)).toBeTruthy()

        tool("Unmute")
        admin.muting = new GetMuteResponse()
        confirm("Unmute", "Yes, unmute")
        expect(await screen.findByText(/^Unmuted\. The mute was until .*\.$/)).toBeTruthy()
        expect(await screen.findByText("Not muted")).toBeTruthy()
        expect(admin.askedFor("unmute")).toEqual([{accountId: ACCOUNT, kind: MuteKind.CHAT, reason: ""}])

        admin.refuse("unmute", Code.NotFound)
        confirm("Unmute", "Yes, unmute")
        expect(await screen.findByText("This player has no mute of its own to lift.")).toBeTruthy()
    })

    it("shows the account's moderation history, why each was done and whether Claude did it", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)

        const history = within(await screen.findByRole("region", {name: "Moderation history"}))
        expect(await history.findByText("GIFs muted for 15 minutes")).toBeTruthy()
        expect(history.getByText("a gore GIF, 2nd offence")).toBeTruthy()
        expect(history.getByText("Claude").className).toContain("badge-app")
        expect(history.getByText("Renamed from Hitler88 to Player42")).toBeTruthy()
        expect(history.getByText("Operator").title).toBe("account-operator")
    })

    it("sends the reason typed with a mute, and reads the history again", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)
        await screen.findByText("GIFs muted for 15 minutes")

        tool("Mute")
        fireEvent.change(screen.getByRole("textbox", {name: "Reason (optional)"}), {target: {value: " spam, 2nd offence "}})
        confirm("Mute", "Yes, mute")

        expect(await screen.findByText(/^Muted until /)).toBeTruthy()
        expect((admin.askedFor("mute") as {reason: string}[])[0].reason).toBe("spam, 2nd offence")
        await waitFor(() => expect(admin.askedFor("listModeration")).toHaveLength(2))
    })

    it("reads no history for an address: the chat and the names are an account's", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={address}/>)

        await screen.findByRole("tab", {name: "Revert"})
        expect(screen.queryByRole("region", {name: "Moderation history"})).toBeNull()
        expect(admin.askedFor("listModeration")).toEqual([])
    })

    it("mutes the GIFs of an account alone, and says so beside the mute of the chat", async () => {
        const admin = new FakeAdmin()
        render(<Page admin={admin} first={account}/>)
        tool("Mute")
        fireEvent.change(screen.getByRole("combobox", {name: "Mute what"}), {target: {value: String(MuteKind.GIFS)}})
        admin.gifMuting = new GetMuteResponse({mutedUntil: Timestamp.fromDate(AT), mutedAccountId: ACCOUNT})
        confirm("Mute", "Yes, mute")

        expect(await screen.findByText(/^GIFs muted until .*\.$/)).toBeTruthy()
        expect((admin.askedFor("mute") as {kind: MuteKind}[])[0].kind).toBe(MuteKind.GIFS)
        expect(await screen.findByText(`GIFs muted until ${timeOf(AT)}`)).toBeTruthy()
        expect(screen.getByText("Not muted")).toBeTruthy()
    })

    it("says when the mute that holds an account was given on its network", async () => {
        const admin = new FakeAdmin()
        admin.muting = new GetMuteResponse({mutedUntil: Timestamp.fromDate(AT), mutedAccountId: "account-household"})
        render(<Page admin={admin} first={account}/>)

        expect(await screen.findByText(`Muted until ${timeOf(AT)}, by a mute on its network`)).toBeTruthy()
    })
})
