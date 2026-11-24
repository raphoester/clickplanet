// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {useState} from "react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {Places} from "../boards/places.ts"
import {timeOf} from "../components/time.ts"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {learnFlags} from "../flags.ts"
import {Flag, RejectionReason, SubmissionState} from "../gen/grpc/flags/v1/flags_pb.ts"
import {patch} from "../graffiti/testPlaces.ts"
import {SubmissionsView} from "./SubmissionsView.tsx"

const DAY = 86_400_000
const BARCELONA = patch(41.3874, 2.1686, 16).places
const readBarcelona = () => Promise.resolve(BARCELONA)

beforeEach(() => {
    vi.useFakeTimers({now: AT, toFake: ["Date"]})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    learnFlags([])
})

function Inbox({admin, first = "", places = readBarcelona}: {
    admin: FakeAdmin, first?: string, places?: (board: string) => Promise<Places>,
}) {
    const [opened, setOpened] = useState(first)
    return (
        <>
            <p data-testid="opened">{opened}</p>
            <SubmissionsView opened={opened} onOpen={setOpened} clients={admin.clients()} places={places}/>
        </>
    )
}

const list = () => within(screen.getByRole("list", {name: "The submissions"}))
const opened = () => screen.getByTestId("opened").textContent
const detail = async () => within(await screen.findByRole("region", {name: "Submission"}))
const verdict = async () => within(await screen.findByRole("region", {name: "Verdict"}))

describe("the submissions inbox", () => {
    it("lists the waiting ones, the oldest first, with each filter's count, and reads each list once", async () => {
        const admin = new FakeAdmin()
        render(<Inbox admin={admin}/>)

        expect(await screen.findByRole("button", {name: "Waiting (3)"})).toBeTruthy()
        expect(screen.getByRole("button", {name: "To open (1)"})).toBeTruthy()
        expect(screen.getByRole("button", {name: "All (6)"})).toBeTruthy()
        const rows = list().getAllByRole("listitem")
        expect(rows.map(row => row.querySelector(".name")?.textContent)).toEqual(["Catalonia", "Lake Garda", "Free State of Bavaria"])
        expect(rows[0].textContent).toContain("New flag · by BraveFox42")
        expect(rows[1].textContent).toContain("Many colours")
        expect(rows[1].textContent).toContain("by account…")
        expect(rows[2].textContent).toContain("Change of Bavaria")
        expect(admin.askedFor("listSubmissions")).toHaveLength(1)
        expect(admin.askedFor("listFlags")).toHaveLength(1)
        expect(admin.askedFor("getPlayers")).toEqual([{accountIds: [ACCOUNT, "account-gone"]}])
        expect(screen.getByText("Pick a submission on the left.")).toBeTruthy()
    })

    it("opens one with its picture as drawn and on a tile, its founder, its pin on the map and the flags near it", async () => {
        const admin = new FakeAdmin()
        const places = vi.fn(() => Promise.resolve(BARCELONA))
        render(<Inbox admin={admin} first="sub-catalonia" places={places}/>)

        const open = await detail()
        expect(open.getByRole("heading", {name: "Catalonia"})).toBeTruthy()
        const picture = open.getByRole("img", {name: "The flag of Catalonia"}) as HTMLImageElement
        expect(picture.getAttribute("src")).toBe("https://bucket.test/flags/submissions/sub-catalonia.png?signed")
        expect(picture.getAttribute("width")).toBe("768")
        expect(open.getByRole("img", {name: "The flag of Catalonia on a tile 50 px wide"}).getAttribute("width")).toBe("50")
        expect(open.getByRole("img", {name: "The flag of Catalonia on a tile 16 px wide"})).toBeTruthy()
        expect(open.queryByText(/most likely a photo/)).toBeNull()
        expect(open.getByText("BraveFox42")).toBeTruthy()
        expect(open.getByText("41.3874, 2.1686", {exact: false})).toBeTruthy()
        expect(open.getByRole("link", {name: "Open on the globe"}).getAttribute("href"))
            .toBe("https://clickplanet.lol/play?lat=41.3874&lon=2.1686&zoom=20")

        const map = await open.findByRole("img", {name: "The land around the pin of Catalonia, north up"})
        expect(map.querySelectorAll("polygon").length).toBeGreaterThan(100)
        expect(map.querySelector(".pin-mark")).toBeTruthy()
        expect(places).toHaveBeenCalledWith("earth-1")
        const near = within(open.getByRole("list", {name: "Flags near it"})).getAllByRole("listitem").map(item => item.textContent?.trim())
        expect(near).toEqual([
            expect.stringMatching(/^Savoy sold, closed\d+ km$/),
            expect.stringMatching(/^Sardinia submitted, approved\d+ km$/),
            expect.stringMatching(/^Brittany submitted, rejected: Has a flag already\d+ km$/),
            expect.stringMatching(/^Lake Garda submitted, waiting\d+ km$/),
        ])
    })

    it("warns of a picture with many colours, and names a founder no player read names by its id", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-beach"/>)

        const open = await detail()
        expect(open.getByRole("alert").textContent).toBe(
            "41% of the picture is in colours past its 8 most common: most likely a photo, not a flag.")
        expect(open.getByText("account-gone")).toBeTruthy()
        expect(open.getByRole("button", {name: "Copy"})).toBeTruthy()
    })

    it("approves once asked, then opens the next waiting one and says what was done", async () => {
        const admin = new FakeAdmin()
        render(<Inbox admin={admin} first="sub-catalonia"/>)

        const panel = await verdict()
        fireEvent.click(panel.getByRole("button", {name: "Approve"}))
        expect(panel.getByRole("group", {name: "Approve"}).textContent)
            .toContain("Approve Catalonia? Its picture becomes public, and the flag waits to be opened.")
        expect(admin.askedFor("approveSubmission")).toEqual([])
        fireEvent.click(panel.getByRole("button", {name: "Yes, approve"}))

        expect((await screen.findByRole("status")).textContent).toBe("Catalonia: Approved.")
        await waitFor(() => expect(opened()).toBe("sub-beach"))
        expect(admin.askedFor("approveSubmission")).toEqual([{submissionId: "sub-catalonia"}])
        expect(screen.getByRole("button", {name: "Waiting (2)"})).toBeTruthy()
        expect(screen.getByRole("button", {name: "To open (2)"})).toBeTruthy()
        expect(list().queryByText("Catalonia")).toBeNull()
    })

    it("rejects with a reason the player reads, and Other only with a note", async () => {
        const admin = new FakeAdmin()
        render(<Inbox admin={admin} first="sub-beach"/>)

        const panel = await verdict()
        const reject = () => panel.getByRole("button", {name: "Reject"}) as HTMLButtonElement
        expect(reject().disabled).toBe(true)
        fireEvent.change(panel.getByLabelText("Reason"), {target: {value: String(RejectionReason.OTHER)}})
        expect(reject().disabled).toBe(true)
        expect(panel.getByText("The player reads the reason and the note.")).toBeTruthy()
        fireEvent.change(panel.getByLabelText(/^Note/), {target: {value: " A photo of the lake. "}})
        expect(reject().disabled).toBe(false)
        fireEvent.click(reject())
        expect(panel.getByRole("group", {name: "Reject"}).textContent)
            .toContain("Reject Lake Garda as “Other”? The player reads this reason and your note.")
        fireEvent.click(panel.getByRole("button", {name: "Yes, reject"}))

        expect((await screen.findByRole("status")).textContent).toBe("Lake Garda: Rejected as “Other”.")
        await waitFor(() => expect(opened()).toBe("sub-bavaria"))
        expect(admin.askedFor("rejectSubmission")).toEqual([{
            submissionId: "sub-beach", rejection: {reason: RejectionReason.OTHER, note: "A photo of the lake."},
        }])
        expect(admin.flags.submissions.find(each => each.id === "sub-beach")?.state).toBe(SubmissionState.REJECTED)
    })

    it("refuses a note past 500 characters as it is typed", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-beach"/>)

        const panel = await verdict()
        fireEvent.change(panel.getByLabelText("Reason"), {target: {value: String(RejectionReason.NOT_A_PLACE)}})
        fireEvent.change(panel.getByLabelText(/^Note/), {target: {value: "a".repeat(501)}})
        expect(panel.getByText("A note of 500 characters at most.")).toBeTruthy()
        expect((panel.getByRole("button", {name: "Reject"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("says the server's refusals, and keeps the submission open", async () => {
        const admin = new FakeAdmin()
        admin.refuse("approveSubmission", Code.FailedPrecondition)
        admin.refuse("rejectSubmission", Code.InvalidArgument)
        render(<Inbox admin={admin} first="sub-catalonia"/>)

        const panel = await verdict()
        fireEvent.click(panel.getByRole("button", {name: "Approve"}))
        fireEvent.click(panel.getByRole("button", {name: "Yes, approve"}))
        expect(await panel.findByText("It is decided already, or the flags' pictures are off on the server.")).toBeTruthy()

        fireEvent.change(panel.getByLabelText("Reason"), {target: {value: String(RejectionReason.UNREADABLE)}})
        fireEvent.click(panel.getByRole("button", {name: "Reject"}))
        fireEvent.click(panel.getByRole("button", {name: "Yes, reject"}))
        expect(await panel.findByText("The server refused it: refused")).toBeTruthy()
        expect(opened()).toBe("sub-catalonia")
        expect(screen.queryByRole("status")).toBeNull()
    })

    it("shows a change beside the flag it changes, and approves it onto that flag", async () => {
        const admin = new FakeAdmin()
        render(<Inbox admin={admin} first="sub-bavaria"/>)

        const open = await detail()
        expect(open.getByText("Change of Bavaria")).toBeTruthy()
        expect(open.getByText("Now: Bavaria")).toBeTruthy()
        expect(open.getByText("Asked: Free State of Bavaria")).toBeTruthy()
        expect(open.getByRole("img", {name: "The flag of Bavaria"}).getAttribute("src")).toBe("https://media.test/flags/pictures/r-k3p9x2ma0b.png")
        expect(open.getByRole("img", {name: "The flag of Free State of Bavaria"})).toBeTruthy()

        const panel = await verdict()
        fireEvent.click(panel.getByRole("button", {name: "Approve"}))
        expect(panel.getByRole("group", {name: "Approve"}).textContent)
            .toContain("Approve the new name and picture of Bavaria? Every player sees them at once.")
        fireEvent.click(panel.getByRole("button", {name: "Yes, approve"}))

        expect((await screen.findByRole("status")).textContent).toBe("Free State of Bavaria: Approved: Bavaria is Free State of Bavaria now.")
        expect(admin.flags.listed.find(flag => flag.code === "r-k3p9x2ma0b")?.name).toBe("Free State of Bavaria")
    })

    it("opens an approved new flag once asked, until its end, and reads the lists again", async () => {
        const admin = new FakeAdmin()
        render(<Inbox admin={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "To open (1)"}))
        fireEvent.click(list().getByRole("button", {name: /Sardinia/}))
        const panel = within(await screen.findByRole("region", {name: "Open the flag"}))
        expect(screen.queryByRole("region", {name: "Verdict"})).toBeNull()
        fireEvent.click(panel.getByRole("button", {name: "Open"}))
        const question = panel.getByRole("group", {name: "Open"}).textContent
        expect(question).toContain(`Open Sardinia until ${timeOf(new Date(AT.getTime() + 30 * DAY))} (30 days left), at its pin?`)
        expect(question).toContain("Every player sees it at once")
        expect(question).toContain("A flag is never deleted")
        expect(admin.askedFor("openSubmittedFlag")).toEqual([])
        fireEvent.click(panel.getByRole("button", {name: "Yes, open"}))

        expect((await screen.findByRole("status")).textContent)
            .toBe(`Sardinia: Opened as r-0000000007, until ${timeOf(new Date(AT.getTime() + 30 * DAY))}.`)
        expect(admin.askedFor("openSubmittedFlag")).toEqual([expect.objectContaining({submissionId: "sub-sardinia"})])
        await waitFor(() => expect(admin.askedFor("listSubmissions")).toHaveLength(2))
        expect(await screen.findByRole("button", {name: "To open (0)"})).toBeTruthy()
        const open = await detail()
        expect(await open.findByText("r-0000000007")).toBeTruthy()
        expect(open.getByText("Opened", {selector: ".badge"})).toBeTruthy()
        expect(screen.queryByRole("region", {name: "Open the flag"})).toBeNull()
    })

    it("opens nothing with an end gone by", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-sardinia"/>)

        const panel = within(await screen.findByRole("region", {name: "Open the flag"}))
        fireEvent.change(panel.getByLabelText("Ends"), {target: {value: "2020-01-01T00:00"}})
        expect(panel.getByText("An end in the future.")).toBeTruthy()
        expect((panel.getByRole("button", {name: "Open"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("shows a rejected one's reason, with no verdict", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-brittany"/>)

        const open = await detail()
        expect(open.getByText("Has a flag already")).toBeTruthy()
        expect(open.getByText("Rejected", {selector: ".badge"})).toBeTruthy()
        expect(screen.queryByRole("region", {name: "Verdict"})).toBeNull()
    })

    it("lists the flags of the same name, the game's own among them", async () => {
        const admin = new FakeAdmin()
        admin.flags.listed = [...admin.flags.listed, new Flag({code: "xb", name: "Brittany"})]
        render(<Inbox admin={admin} first="sub-brittany"/>)

        const same = within(await screen.findByRole("region", {name: "The same name"}))
        expect(same.getByText("the game's own")).toBeTruthy()
    })

    it("says when nothing waits", async () => {
        const admin = new FakeAdmin()
        admin.flags.submissions = []
        render(<Inbox admin={admin}/>)

        expect(await screen.findByText("No submission waits. All done.")).toBeTruthy()
    })

    it("says the map could not be drawn on a board the page does not carry, and shows the rest", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-catalonia" places={() => Promise.reject(new Error("earth-9 is not a board"))}/>)

        const open = await detail()
        expect(await open.findByText("The map could not be drawn: earth-9 is not a board")).toBeTruthy()
        expect(open.getByRole("list", {name: "Flags near it"})).toBeTruthy()
    })
})
