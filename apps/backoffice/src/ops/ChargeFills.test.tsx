// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {act, cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {GetChargeFillsResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {ChargeFills, FOLLOW_MS} from "./ChargeFills.tsx"

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

const statusLine = () => screen.findByText((_, element) => element?.tagName === "P" &&
    /^(Last fill:|No fill yet\.)/.test(element.textContent ?? ""))

describe("the charge fills", () => {
    it("says the last fill, how far it got, and who started it", async () => {
        render(<ChargeFills planet={new FakeAdmin()}/>)

        expect((await statusLine()).textContent).toBe("Last fill: Done, 1200 accounts filled")
        expect(screen.getByText("season-0:finale:fill")).toBeTruthy()
        expect(screen.getByText("Seasons")).toBeTruthy()
    })

    it("says when there is none yet", async () => {
        const admin = new FakeAdmin()
        admin.chargeFills = new GetChargeFillsResponse()
        render(<ChargeFills planet={admin}/>)

        expect((await statusLine()).textContent).toBe("No fill yet.")
        expect(screen.queryByRole("table")).toBeNull()
    })

    it("fills every bank once confirmed, under a request of its own, and follows it until it is done", async () => {
        vi.useFakeTimers({shouldAdvanceTime: true})
        const admin = new FakeAdmin()
        render(<ChargeFills planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Fill every bank"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, fill every bank"}))

        await waitFor(async () => expect((await statusLine()).textContent).toBe("Last fill: Running, 0 accounts filled"))
        const [asked] = admin.askedFor("fillCharges") as {requestId: string}[]
        expect(asked.requestId).not.toBe("")
        expect(screen.getByText("An operator")).toBeTruthy()

        admin.finishFill(1300)
        await act(() => vi.advanceTimersByTimeAsync(FOLLOW_MS))

        await waitFor(async () => expect((await statusLine()).textContent).toBe("Last fill: Done, 1300 accounts filled"))
    })

    it("says a refusal in plain words", async () => {
        const admin = new FakeAdmin()
        admin.refuse("fillCharges", Code.Unauthenticated)
        render(<ChargeFills planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Fill every bank"}))
        fireEvent.click(screen.getByRole("button", {name: "Yes, fill every bank"}))

        expect(await screen.findByText("Your session ended. Reload the page and sign in again.")).toBeTruthy()
    })
})
