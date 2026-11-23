// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {localOf, timeOf} from "../components/time.ts"
import {AT, FakeAdmin, soldFlag} from "../fakeAdmin.ts"
import {learnFlags, listedFlag} from "../flags.ts"
import {Put} from "../videos/actions.ts"
import {FlagsPage} from "./FlagsPage.tsx"

const DAY = 86_400_000
const later = (days: number) => new Date(AT.getTime() + days * DAY)

beforeEach(() => {
    vi.useFakeTimers({now: AT, toFake: ["Date"]})
    URL.createObjectURL = () => "blob:picked"
    URL.revokeObjectURL = () => {}
})

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    learnFlags([])
})

// A PUT the fake records among the calls, so their order shows.
function putting(admin: FakeAdmin) {
    return vi.fn<Put>((url, file, _type, onSent) => {
        admin.asked.push({method: "put", request: url})
        onSent(file.size)
        return Promise.resolve()
    })
}

const card = async (name: string) => (await screen.findByRole("heading", {name})).closest("li") as HTMLElement
const names = () => within(screen.getByRole("region", {name: "Sold flags"})).getAllByRole("heading").map(h => h.textContent)
const picture = () => new File(["0123456789"], "rennes.png", {type: "image/png"})

function fill(name: string, ends: Date) {
    const form = screen.getByRole("region", {name: "Open a flag"})
    fireEvent.change(within(form).getByLabelText(/^Name/), {target: {value: name}})
    fireEvent.change(within(form).getByLabelText("Ends"), {target: {value: localOf(ends)}})
    fireEvent.change(within(form).getByLabelText("Pick a picture"), {target: {files: [picture()]}})
    return form
}

describe("FlagsPage", () => {
    it("lists the sold flags alone, each open one with its days left and each closed one with its comeback", async () => {
        const admin = new FakeAdmin()
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)

        const bavaria = await card("Bavaria")
        expect(within(bavaria).getByText("Open", {selector: ".status"})).toBeTruthy()
        expect(within(bavaria).getByText(/12 days left/)).toBeTruthy()
        expect(within(bavaria).getByText(timeOf(later(12)))).toBeTruthy()
        expect(within(bavaria).getByText("r-k3p9x2ma0b")).toBeTruthy()
        expect(within(bavaria).getByRole("img", {name: "The flag of Bavaria"}).getAttribute("src"))
            .toBe("https://media.test/flags/pictures/r-k3p9x2ma0b.png")

        const savoy = await card("Savoy")
        expect(within(savoy).getByText("Closed", {selector: ".status"})).toBeTruthy()
        expect(within(savoy).getByText(timeOf(later(-3)))).toBeTruthy()
        expect(within(savoy).getByText(timeOf(later(27)))).toBeTruthy()
        expect((within(savoy).getByRole("button", {name: "Close"}) as HTMLButtonElement).disabled).toBe(true)

        expect(screen.getByText("2 sold flags, 1 open")).toBeTruthy()
        expect(screen.queryByRole("heading", {name: "France"})).toBeNull()
    })

    it("sorts the open ones by end and the closed ones after, or all by name", async () => {
        const admin = new FakeAdmin()
        admin.flags.listed = [...admin.flags.listed, soldFlag("r-alsace0001", "Alsace", later(20))]
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)
        await card("Alsace")

        expect(names()).toEqual(["Bavaria", "Alsace", "Savoy"])
        fireEvent.click(screen.getByRole("button", {name: "By name"}))
        expect(names()).toEqual(["Alsace", "Bavaria", "Savoy"])
    })

    it("names a sold flag to the rest of the console once read", async () => {
        const admin = new FakeAdmin()
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)
        await card("Bavaria")

        expect(listedFlag("r-k3p9x2ma0b")).toEqual({
            code: "r-k3p9x2ma0b", name: "Bavaria", closed: false, picture: "https://media.test/flags/pictures/r-k3p9x2ma0b.png",
        })
    })

    it("asks first, then uploads the picture, opens the flag with it and reads the list again", async () => {
        const admin = new FakeAdmin()
        const put = putting(admin)
        render(<FlagsPage flags={admin.flags} put={put}/>)
        await card("Bavaria")

        const form = fill(" Rennes ", later(20))
        expect(within(form).getByText("rennes.png · 10 B")).toBeTruthy()
        fireEvent.click(within(form).getByRole("button", {name: "Open"}))

        const question = within(form).getByRole("group", {name: "Open"})
        expect(question.textContent).toContain(`Open Rennes until ${timeOf(later(20))} (20 days left)?`)
        expect(question.textContent).toContain("Every player sees it at once")
        expect(question.textContent).toContain("A flag is never deleted")
        expect(admin.askedFor("createFlagUpload")).toEqual([])

        fireEvent.click(within(form).getByRole("button", {name: "Yes, open"}))

        expect(await within(form).findByText(`Opened Rennes (r-0000000007), until ${timeOf(later(20))}.`)).toBeTruthy()
        expect(await card("Rennes")).toBeTruthy()
        expect(admin.asked.map(call => call.method)).toEqual(["listFlags", "createFlagUpload", "put", "openFlag", "listFlags"])
        expect(admin.askedFor("createFlagUpload")).toEqual([{contentType: "image/png", sizeBytes: 10n}])
        expect(put).toHaveBeenCalledWith("https://bucket.test/flags/uploads/flag-upload-1", expect.any(File), "image/png", expect.any(Function))
        expect(admin.askedFor("openFlag")).toEqual([expect.objectContaining({name: "Rennes", uploadId: "flag-upload-1"})])
        expect((admin.askedFor("openFlag")[0] as {endsAt: {toDate: () => Date}}).endsAt.toDate()).toEqual(later(20))
        expect((within(form).getByLabelText(/^Name/) as HTMLInputElement).value).toBe("")
    })

    it("opens nothing with a name too long or an end gone by", async () => {
        const admin = new FakeAdmin()
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)
        await card("Bavaria")

        const form = fill("A".repeat(41), later(-1))

        expect(within(form).getByText("A name of 40 characters at most.")).toBeTruthy()
        expect(within(form).getByText("An end in the future.")).toBeTruthy()
        expect((within(form).getByRole("button", {name: "Open"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("says the server's refusal of the flag", async () => {
        const admin = new FakeAdmin()
        admin.refuse("openFlag", Code.InvalidArgument)
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)
        await card("Bavaria")

        const form = fill("Rennes", later(20))
        fireEvent.click(within(form).getByRole("button", {name: "Open"}))
        fireEvent.click(within(form).getByRole("button", {name: "Yes, open"}))

        expect(await within(form).findByText("The server refused it: refused")).toBeTruthy()
        expect(screen.queryByRole("heading", {name: "Rennes"})).toBeNull()
    })

    it("sends no picture when the server has the pictures off", async () => {
        const admin = new FakeAdmin()
        admin.refuse("createFlagUpload", Code.FailedPrecondition)
        const put = putting(admin)
        render(<FlagsPage flags={admin.flags} put={put}/>)
        await card("Bavaria")

        const form = fill("Rennes", later(20))
        fireEvent.click(within(form).getByRole("button", {name: "Open"}))
        fireEvent.click(within(form).getByRole("button", {name: "Yes, open"}))

        expect(await within(form).findByText("The flags' pictures are off on the server.")).toBeTruthy()
        expect(put).not.toHaveBeenCalled()
        expect(admin.askedFor("openFlag")).toEqual([])
    })

    it("adds days to an open flag once asked, says its new end and reads the list again", async () => {
        const admin = new FakeAdmin()
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)
        const bavaria = await card("Bavaria")

        fireEvent.change(within(bavaria).getByLabelText("Days"), {target: {value: "10"}})
        fireEvent.click(within(bavaria).getByRole("button", {name: "Add days"}))
        expect(within(bavaria).getByRole("group", {name: "Add days"}).textContent)
            .toContain(`Add 10 days to Bavaria? It ends on ${timeOf(later(22))} instead of ${timeOf(later(12))}.`)
        expect(admin.askedFor("addTime")).toEqual([])
        fireEvent.click(within(bavaria).getByRole("button", {name: "Yes, add days"}))

        expect(await within(bavaria).findByText(`It ends on ${timeOf(later(22))}.`)).toBeTruthy()
        expect(admin.askedFor("addTime")).toEqual([{code: "r-k3p9x2ma0b", days: 10}])
        expect(await within(bavaria).findByText(/22 days left/)).toBeTruthy()
        expect(admin.askedFor("listFlags")).toHaveLength(2)
    })

    it("opens a flag closed less than 30 days ago again", async () => {
        const admin = new FakeAdmin()
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)
        const savoy = await card("Savoy")

        fireEvent.click(within(savoy).getByRole("button", {name: "Add days"}))
        expect(within(savoy).getByRole("group", {name: "Add days"}).textContent)
            .toContain(`Open Savoy again for 30 days, until ${timeOf(later(30))}?`)
        fireEvent.click(within(savoy).getByRole("button", {name: "Yes, add days"}))

        expect(await within(savoy).findByText(`Open again. It ends on ${timeOf(later(30))}.`)).toBeTruthy()
        expect(await within(savoy).findByText(/30 days left/)).toBeTruthy()
        expect(within(savoy).getByText("Open", {selector: ".status"})).toBeTruthy()
    })

    it("adds no days to a flag closed over 30 days ago", async () => {
        const admin = new FakeAdmin()
        admin.flags.listed = [soldFlag("r-tyrol00001", "Tyrol", later(-40), later(-40))]
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)
        const tyrol = await card("Tyrol")

        expect(within(tyrol).getByText("no more: closed over 30 days ago")).toBeTruthy()
        expect((within(tyrol).getByRole("button", {name: "Add days"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("closes an open flag once asked, and reads the list again", async () => {
        const admin = new FakeAdmin()
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)
        const bavaria = await card("Bavaria")

        fireEvent.click(within(bavaria).getByRole("button", {name: "Close"}))
        expect(within(bavaria).getByRole("group", {name: "Close"}).textContent).toContain("Its players must pick another flag")
        expect(admin.askedFor("closeFlag")).toEqual([])
        fireEvent.click(within(bavaria).getByRole("button", {name: "Yes, close"}))

        expect(await within(bavaria).findByText(`Closed. Time added until ${timeOf(later(30))} opens it again.`)).toBeTruthy()
        expect(admin.askedFor("closeFlag")).toEqual([{code: "r-k3p9x2ma0b"}])
        expect(await within(bavaria).findByText("Closed", {selector: ".status"})).toBeTruthy()
        expect(admin.askedFor("listFlags")).toHaveLength(2)
    })

    it("says the server's refusal of a close", async () => {
        const admin = new FakeAdmin()
        admin.refuse("closeFlag", Code.FailedPrecondition)
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)
        const bavaria = await card("Bavaria")

        fireEvent.click(within(bavaria).getByRole("button", {name: "Close"}))
        fireEvent.click(within(bavaria).getByRole("button", {name: "Yes, close"}))

        expect(await within(bavaria).findByText("It is closed already.")).toBeTruthy()
        expect(admin.askedFor("listFlags")).toHaveLength(1)
    })

    it("says when the list cannot be read", async () => {
        const admin = new FakeAdmin()
        admin.refuse("listFlags", Code.Unavailable)
        render(<FlagsPage flags={admin.flags} put={putting(admin)}/>)

        expect(await screen.findByText("The call failed.")).toBeTruthy()
    })
})
