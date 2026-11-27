// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, waitFor, within} from "@testing-library/react"
import {useState} from "react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {Timestamp} from "@bufbuild/protobuf"
import {timeOf} from "../components/time.ts"
import {ACCOUNT, AT, FakeAdmin, keptSubmission, soldFlag} from "../fakeAdmin.ts"
import {learnFlags} from "../flags.ts"
import {placeWords} from "../flags/pin.ts"
import {Judgement, JudgementVerdict} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Appeal, Flag, Rejection, RejectionReason, SubmissionState} from "../gen/grpc/flags/v1/flags_pb.ts"
import {Letter, LetterMail, NoteReply} from "../gen/grpc/support/v1/admin_pb.ts"
import {patch} from "../graffiti/testPlaces.ts"
import {placeAt, regionOf, roundedOf} from "./mapView.ts"
import {Ground} from "./submissions.ts"
import {SubmissionsView} from "./SubmissionsView.tsx"

const DAY = 86_400_000
const CATALONIA = {latitude: 41.3874, longitude: 2.1686}
const BARCELONA: Ground = {
    places: patch(CATALONIA.latitude, CATALONIA.longitude, 16).places,
    capitals: [
        {country: "ad", latitude: 42.5063, longitude: 1.5218},
        {country: "es", latitude: 40.4168, longitude: -3.7038},
        {country: "fr", latitude: 48.8566, longitude: 2.3522},
    ],
}
const readBarcelona = () => Promise.resolve(BARCELONA)
// jsdom lays nothing out: the map is 640 px wide, 320 high, and a pointer's place in it is its client's.
const REGION = regionOf(CATALONIA, 640, 320)

beforeEach(() => {
    vi.useFakeTimers({now: AT, toFake: ["Date"]})
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    learnFlags([])
})

function Inbox({admin, first = "", ground = readBarcelona}: {
    admin: FakeAdmin, first?: string, ground?: (board: string) => Promise<Ground>,
}) {
    const [opened, setOpened] = useState(first)
    return (
        <>
            <p data-testid="opened">{opened}</p>
            <SubmissionsView opened={opened} onOpen={setOpened} clients={admin.clients()} ground={ground}/>
        </>
    )
}

const minute = (n: number) => Timestamp.fromDate(new Date(AT.getTime() + n * 60_000))
// BraveFox42 replied to the letter about Catalonia; Free State of Bavaria is theirs too.
const cataloniaReplied = () => new Letter({
    id: "letter-7", submissionId: "sub-catalonia", accountId: ACCOUNT, address: "ada@example.com", subject: "Your flag Catalonia",
    mails: [new LetterMail({text: "Which Catalonia?", sentAt: minute(-30)})],
    replies: [new NoteReply({text: "the spanish one", from: "ada@example.com", receivedAt: minute(-10)})],
})

const list = () => within(screen.getByRole("list", {name: "The submissions"}))
const opened = () => screen.getByTestId("opened").textContent
const detail = async () => within(await screen.findByRole("region", {name: "Submission"}))
const verdict = async () => within(await screen.findByRole("region", {name: "Verdict"}))
const map = async (name = "Catalonia") =>
    (await screen.findByRole("img", {name: `The capital of ${name} on a map of the world`})).closest(".capital-map") as HTMLElement
const ownMark = (frame: HTMLElement) => frame.querySelector(".mark-own") as HTMLElement
const caption = (frame: HTMLElement) => frame.closest("figure")?.querySelector("figcaption")?.textContent
const tap = (frame: HTMLElement, x: number, y: number) => {
    fireEvent.pointerDown(frame, {pointerId: 1, clientX: x, clientY: y})
    fireEvent.pointerUp(frame, {pointerId: 1, clientX: x, clientY: y})
}

describe("the submissions inbox", () => {
    it("lists the waiting ones, the oldest first, with each filter's count, and reads each list once", async () => {
        const admin = new FakeAdmin()
        render(<Inbox admin={admin}/>)

        expect(await screen.findByRole("button", {name: "Waiting (3)"})).toBeTruthy()
        expect(screen.getByRole("button", {name: "To open (1)"})).toBeTruthy()
        expect(screen.getByRole("button", {name: "All (6)"})).toBeTruthy()
        const rows = list().getAllByRole("listitem")
        expect(rows.map(row => row.querySelector(".name")?.textContent)).toEqual(["Catalonia", "Lake Garda", "Free State of Bavaria"])
        expect(rows[0].textContent).toContain("by BraveFox42")
        expect(rows[1].textContent).toContain("Many colours")
        expect(rows[1].textContent).toContain("by account…")
        expect(admin.askedFor("listSubmissions")).toHaveLength(1)
        expect(admin.askedFor("listFlags")).toHaveLength(1)
        expect(admin.askedFor("getPlayers")).toEqual([{accountIds: [ACCOUNT, "account-gone"]}])
        expect(screen.getByText("Pick a submission on the left.")).toBeTruthy()
    })

    it("opens one with its picture as drawn and on a tile, its founder, its capital on the map and the capitals near it", async () => {
        const admin = new FakeAdmin()
        const ground = vi.fn(readBarcelona)
        render(<Inbox admin={admin} first="sub-catalonia" ground={ground}/>)

        const open = await detail()
        expect(open.getByRole("heading", {name: "Catalonia"})).toBeTruthy()
        const picture = open.getByRole("img", {name: "The flag of Catalonia"}) as HTMLImageElement
        expect(picture.getAttribute("src")).toBe("https://bucket.test/flags/submissions/sub-catalonia.png?signed")
        expect(picture.getAttribute("width")).toBe("768")
        expect(open.getByRole("img", {name: "The flag of Catalonia on a tile 50 px wide"}).getAttribute("width")).toBe("50")
        expect(open.getByRole("img", {name: "The flag of Catalonia on a tile 16 px wide"})).toBeTruthy()
        expect(open.queryByText(/most likely a photo/)).toBeNull()
        expect(open.getByText("BraveFox42")).toBeTruthy()
        expect(open.getByText("Capital", {selector: "dt"})).toBeTruthy()
        expect(open.getByText("41.3874, 2.1686", {exact: false})).toBeTruthy()
        expect(open.getByRole("link", {name: "Open on the globe"}).getAttribute("href"))
            .toBe("https://clickplanet.lol/play?lat=41.3874&lon=2.1686&zoom=20")

        const frame = await map()
        expect(ground).toHaveBeenCalledWith("earth-1")
        expect(caption(frame)).toMatch(/^earth-1 · 2,0\d\d km across$/)
        expect(ownMark(frame).style.left).toBe("320px")
        expect(ownMark(frame).style.top).toBe("160px")
        expect(frame.querySelector(".label-own")?.textContent?.trim()).toBe("Catalonia")
        // Paris is 830 km north: out of a frame 2,000 km across and half as high.
        expect([...frame.querySelectorAll(".mark-country")].map(mark => mark.getAttribute("title"))).toEqual(["Andorra", "Spain"])
        expect([...frame.querySelectorAll(".map-label")].map(label => label.textContent?.trim()))
            .toEqual(expect.arrayContaining(["🇦🇩 Andorra", "🇪🇸 Spain", "Sardinia", "Savoy"]))
        const near = within(open.getByRole("list", {name: "Capitals near it"})).getAllByRole("listitem").map(item => item.textContent?.trim())
        expect(near).toEqual([
            expect.stringMatching(/^🇦🇩 Andorra the game's own135 km$/),
            expect.stringMatching(/^🇪🇸 Spain the game's own505 km$/),
            expect.stringMatching(/^Savoy sold, closed554 km$/),
            expect.stringMatching(/^Sardinia submitted, approved637 km$/),
            expect.stringMatching(/^Brittany submitted, rejected: Has a flag already807 km$/),
            expect.stringMatching(/^Lake Garda submitted, waiting825 km$/),
            expect.stringMatching(/^🇫🇷 France the game's own831 km$/),
        ])
    })

    it("zooms out to the whole world and back, with the buttons, the wheel and two fingers", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-catalonia"/>)

        const frame = await map()
        const km = () => Number(caption(frame)?.match(/([\d,]+) km across/)?.[1].replace(",", ""))
        const region = km()
        fireEvent.click(within(frame).getByRole("button", {name: "World"}))
        expect(caption(frame)).toBe("earth-1 · the whole world")
        expect(frame.querySelectorAll(".mark-country")).toHaveLength(3)
        expect([...frame.querySelectorAll(".map-label")].map(label => label.textContent?.trim())).toEqual(["Catalonia"])
        fireEvent.click(within(frame).getByRole("button", {name: "Zoom out"}))
        expect(caption(frame)).toBe("earth-1 · the whole world")

        fireEvent.click(within(frame).getByRole("button", {name: "Capital"}))
        expect(km()).toBe(region)
        fireEvent.click(within(frame).getByRole("button", {name: "Zoom in"}))
        expect(km()).toBeCloseTo(region / 2, -1)
        fireEvent.wheel(frame, {deltaY: 200, clientX: 320, clientY: 160})
        expect(km()).toBeGreaterThan(region / 2)

        fireEvent.click(within(frame).getByRole("button", {name: "Capital"}))
        fireEvent.pointerDown(frame, {pointerId: 1, clientX: 300, clientY: 160})
        fireEvent.pointerDown(frame, {pointerId: 2, clientX: 340, clientY: 160})
        fireEvent.pointerMove(frame, {pointerId: 2, clientX: 380, clientY: 160})
        fireEvent.pointerUp(frame, {pointerId: 2, clientX: 380, clientY: 160})
        fireEvent.pointerUp(frame, {pointerId: 1, clientX: 300, clientY: 160})
        expect(km()).toBeCloseTo(region / 2, -1)
        expect(screen.queryByText(/^Moved to/)).toBeNull()
    })

    it("pans with a drag, and leaves the capital where it is", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-catalonia"/>)

        const frame = await map()
        fireEvent.pointerDown(frame, {pointerId: 1, clientX: 100, clientY: 100})
        fireEvent.pointerMove(frame, {pointerId: 1, clientX: 160, clientY: 130})
        fireEvent.pointerUp(frame, {pointerId: 1, clientX: 160, clientY: 130})
        expect(ownMark(frame).style.left).toBe("380px")
        expect(ownMark(frame).style.top).toBe("190px")
        expect(screen.queryByText(/^Moved to/)).toBeNull()
    })

    it("moves the capital to a tap, says where, approves it there, and goes back to the player's", async () => {
        const admin = new FakeAdmin()
        render(<Inbox admin={admin} first="sub-catalonia"/>)

        const frame = await map()
        const moved = roundedOf(placeAt(REGION, 384, 128))
        tap(frame, 384, 128)
        const open = await detail()
        expect(open.getByText(placeWords(moved)).closest("div")?.textContent).toContain(`Moved to ${placeWords(moved)}`)
        expect(ownMark(frame).style.left).toBe("384px")
        expect(open.getByText("Capitals near it")).toBeTruthy()

        fireEvent.click(open.getByRole("button", {name: "Back to the player's"}))
        expect(open.queryByText(/^Moved to/)).toBeNull()
        expect(ownMark(frame).style.left).toBe("320px")

        tap(frame, 384, 128)
        const panel = await verdict()
        fireEvent.click(panel.getByRole("button", {name: "Approve"}))
        expect(panel.getByRole("group", {name: "Approve"}).textContent).toContain(
            `Approve Catalonia, its capital moved to ${placeWords(moved)}? Its picture becomes public, and the flag waits to be opened.`)
        fireEvent.click(panel.getByRole("button", {name: "Yes, approve"}))

        expect((await screen.findByRole("status")).textContent).toBe(`Catalonia: Approved. Its capital is at ${placeWords(moved)}.`)
        expect(admin.askedFor("approveSubmission")).toEqual([{submissionId: "sub-catalonia", pin: moved}])
        const approved = admin.flags.submissions.find(each => each.id === "sub-catalonia")
        expect({latitude: approved?.pin?.latitude, longitude: approved?.pin?.longitude}).toEqual(moved)
    })

    it("drags the capital's mark", async () => {
        const admin = new FakeAdmin()
        render(<Inbox admin={admin} first="sub-catalonia"/>)

        const frame = await map()
        fireEvent.pointerDown(ownMark(frame), {pointerId: 1, clientX: 322, clientY: 161})
        fireEvent.pointerMove(frame, {pointerId: 1, clientX: 300, clientY: 201})
        fireEvent.pointerUp(frame, {pointerId: 1, clientX: 300, clientY: 201})
        const moved = roundedOf(placeAt(REGION, 298, 200))
        expect((await detail()).getByText(placeWords(moved))).toBeTruthy()
        expect(Number.parseFloat(ownMark(frame).style.left)).toBeCloseTo(298, 1)
        expect(Number.parseFloat(ownMark(frame).style.top)).toBeCloseTo(200, 1)
    })

    it("shows a decided one's capital, which no tap moves", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-brittany"/>)

        const frame = await map("Brittany")
        tap(frame, 384, 128)
        expect(screen.queryByText(/^Moved to/)).toBeNull()
        expect(ownMark(frame).style.left).toBe("320px")
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
        expect(await panel.findByText("The server refused it: refused")).toBeTruthy()

        fireEvent.change(panel.getByLabelText("Reason"), {target: {value: String(RejectionReason.UNREADABLE)}})
        fireEvent.click(panel.getByRole("button", {name: "Reject"}))
        fireEvent.click(panel.getByRole("button", {name: "Yes, reject"}))
        await waitFor(() => expect(panel.getAllByText("The server refused it: refused")).toHaveLength(2))
        expect(opened()).toBe("sub-catalonia")
        expect(screen.queryByRole("status")).toBeNull()
    })

    it("says what the server answered when another flag took the name", async () => {
        const admin = new FakeAdmin()
        admin.flags.listed = [...admin.flags.listed, soldFlag("r-catalonia0", "Catalonia", new Date(AT.getTime() + DAY))]
        render(<Inbox admin={admin} first="sub-catalonia"/>)

        const panel = await verdict()
        fireEvent.click(panel.getByRole("button", {name: "Approve"}))
        fireEvent.click(panel.getByRole("button", {name: "Yes, approve"}))
        expect(await panel.findByText("The server refused it: the flag r-catalonia0 is named Catalonia")).toBeTruthy()
        expect(opened()).toBe("sub-catalonia")
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
        expect(question).toContain(`Open Sardinia until ${timeOf(new Date(AT.getTime() + 30 * DAY))} (30 days left), at its capital?`)
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

    it("marks a flag its player appealed, and shows Claude's rejection it asks about", async () => {
        const admin = new FakeAdmin()
        admin.flags.submissions = [...admin.flags.submissions, keptSubmission("sub-smurfs", "Smurfland", CATALONIA, {
            judgement: new Judgement({verdict: JudgementVerdict.REJECT, why: "a cartoon's village"}),
            appeal: new Appeal({
                askedAt: minute(-10), rejection: new Rejection({reason: RejectionReason.NOT_A_PLACE, note: "A cartoon."}),
            }),
        })]
        render(<Inbox admin={admin} first="sub-smurfs"/>)

        const open = await detail()
        expect(list().getByRole("button", {name: /Smurfland/}).textContent).toContain("Appeal")
        expect(open.getByText(/a cartoon's village/)).toBeTruthy()
        expect(open.getByText(/of Claude's rejection: Not a place's name/)).toBeTruthy()
        expect(open.getByText("A cartoon.")).toBeTruthy()
        expect(await screen.findByRole("region", {name: "Verdict"})).toBeTruthy()
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

    it("writes to the founder: the submission's letter under the page's subject, asked first, then one more mail in it", async () => {
        const admin = new FakeAdmin()
        render(<Inbox admin={admin} first="sub-catalonia"/>)

        const panel = within(await screen.findByRole("region", {name: "Write to BraveFox42"}))
        const letter = within(await panel.findByRole("article", {name: "Letter: Your flag Catalonia"}))
        expect(panel.queryByLabelText(/^Subject/)).toBeNull()
        expect(panel.queryByText("New letter")).toBeNull()
        expect((letter.getByRole("button", {name: "Send"}) as HTMLButtonElement).disabled).toBe(true)
        fireEvent.change(letter.getByLabelText("Text"), {target: {value: "Is it the Spanish region or all the lands?"}})
        fireEvent.click(letter.getByRole("button", {name: "Send"}))
        expect(letter.getByRole("group", {name: "Send"}).textContent).toContain("Send this to BraveFox42's verified address?")
        expect(admin.askedFor("writeLetter")).toEqual([])
        fireEvent.click(letter.getByRole("button", {name: "Yes, send"}))

        expect(await letter.findByText("Sent to ada@example.com.")).toBeTruthy()
        expect(letter.getByText("To ada@example.com")).toBeTruthy()
        expect(letter.getByText("Is it the Spanish region or all the lands?")).toBeTruthy()

        fireEvent.change(letter.getByLabelText("Write again"), {target: {value: "Thanks, approved."}})
        fireEvent.click(letter.getByRole("button", {name: "Send"}))
        expect(letter.getByRole("group", {name: "Send"}).textContent).toContain("Send this to ada@example.com?")
        fireEvent.click(letter.getByRole("button", {name: "Yes, send"}))
        expect(await letter.findByText("Thanks, approved.")).toBeTruthy()
        expect(admin.askedFor("writeLetter")).toEqual([
            {submissionId: "sub-catalonia", accountId: ACCOUNT, subject: "Your flag Catalonia", text: "Is it the Spanish region or all the lands?"},
            {letterId: "letter-1", text: "Thanks, approved."},
        ])
        expect(admin.askedFor("listLetters")).toContainEqual({submissionId: "sub-catalonia", limit: 1})
    })

    it("marks the submission whose founder replied to its letter, lists it under Replies, and closes the letter", async () => {
        const admin = new FakeAdmin()
        admin.letters = [cataloniaReplied()]
        render(<Inbox admin={admin}/>)

        await screen.findByRole("list", {name: "The submissions"})
        const row = await list().findByRole("button", {name: /Catalonia/})
        expect(within(row).getByText("Replied")).toBeTruthy()
        expect(list().getByRole("button", {name: /Lake Garda/}).textContent).not.toContain("Replied")
        expect(admin.askedFor("listLetters")).toEqual([{openOnly: true, limit: 500}])
        fireEvent.click(screen.getByRole("button", {name: "Replies (1)"}))
        fireEvent.click(list().getByRole("button", {name: /Catalonia/}))

        const letter = within(await screen.findByRole("article", {name: "Letter: Your flag Catalonia"}))
        expect(letter.getByText("the spanish one")).toBeTruthy()
        fireEvent.click(letter.getByRole("button", {name: "Close"}))
        fireEvent.click(letter.getByRole("button", {name: "Yes, close"}))
        expect(await letter.findByText("Closed with no further mail")).toBeTruthy()
        expect(letter.queryByText("Replied")).toBeNull()
        expect(await screen.findByRole("button", {name: "Replies (0)"})).toBeTruthy()
        expect(admin.askedFor("closeLetter")).toEqual([{letterId: "letter-7"}])
    })

    it("shows a submission's own letter alone: the founder's reply about another submission neither shows nor marks it", async () => {
        const admin = new FakeAdmin()
        admin.letters = [cataloniaReplied()]
        render(<Inbox admin={admin}/>)

        await screen.findByRole("list", {name: "The submissions"})
        const bavaria = await list().findByRole("button", {name: /Free State of Bavaria/})
        expect(within(bavaria).queryByText("Replied")).toBeNull()
        fireEvent.click(bavaria)
        const panel = within(await screen.findByRole("region", {name: "Write to BraveFox42"}))
        const letter = within(await panel.findByRole("article", {name: "Letter: Your flag Free State of Bavaria"}))
        expect(panel.queryByText("the spanish one")).toBeNull()
        expect(panel.queryByText("Which Catalonia?")).toBeNull()
        expect(panel.queryByRole("article", {name: "Letter: Your flag Catalonia"})).toBeNull()

        fireEvent.change(letter.getByLabelText("Text"), {target: {value: "Which Bavaria?"}})
        fireEvent.click(letter.getByRole("button", {name: "Send"}))
        fireEvent.click(letter.getByRole("button", {name: "Yes, send"}))
        expect(await letter.findByText("Sent to ada@example.com.")).toBeTruthy()
        expect(admin.letters.map(each => each.submissionId).sort()).toEqual(["sub-bavaria", "sub-catalonia"])
    })

    it("says when a founder has no verified address to write to", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-beach"/>)

        const panel = within(await screen.findByRole("region", {name: "Write to account…"}))
        fireEvent.change(panel.getByLabelText("Text"), {target: {value: "Is this a flag?"}})
        fireEvent.click(panel.getByRole("button", {name: "Send"}))
        fireEvent.click(panel.getByRole("button", {name: "Yes, send"}))
        expect(await panel.findByText("No mail can go: the player has no verified address, or the letter is over 90 days old.")).toBeTruthy()
    })

    it("says the map could not be drawn on a board the page does not carry, and shows the rest", async () => {
        render(<Inbox admin={new FakeAdmin()} first="sub-catalonia" ground={() => Promise.reject(new Error("earth-9 is not a board"))}/>)

        const open = await detail()
        expect(await open.findByText("The map could not be drawn: earth-9 is not a board")).toBeTruthy()
        expect(open.getByRole("list", {name: "Capitals near it"})).toBeTruthy()
    })
})
