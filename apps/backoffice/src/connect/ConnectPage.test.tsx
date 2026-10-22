// @vitest-environment jsdom
import {Code, ConnectError} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {ApproveConnectionResponse, DescribeConnectionResponse} from "../gen/grpc/auth/v1/admin_pb.ts"
import {ConnectAdmin, ConnectPage} from "./ConnectPage.tsx"

afterEach(cleanup)

function admin(describe: () => Promise<DescribeConnectionResponse>): ConnectAdmin & {approved: string[]} {
    const approved: string[] = []
    return {
        approved,
        describeConnection: vi.fn(describe),
        approveConnection: vi.fn(({request}) => {
            approved.push(request ?? "")
            return Promise.resolve(new ApproveConnectionResponse({redirectUrl: "https://claude.ai/api/mcp/auth_callback?code=the-code"}))
        }),
    } as unknown as ConnectAdmin & {approved: string[]}
}

const claude = () => Promise.resolve(new DescribeConnectionResponse({
    clientName: "Claude",
    redirectHost: "claude.ai",
    denyUrl: "https://claude.ai/api/mcp/auth_callback?error=access_denied",
}))

describe("ConnectPage", () => {
    it("says who asks and where it goes, and lets nothing in before Allow", async () => {
        const auth = admin(claude)
        const leave = vi.fn()
        render(<ConnectPage request="sealed" auth={auth} leave={leave} onDismiss={vi.fn()}/>)

        expect(await screen.findByRole("heading", {name: "Connect Claude"})).toBeTruthy()
        expect(screen.getByText("claude.ai")).toBeTruthy()
        expect(auth.approved).toEqual([])
        expect(leave).not.toHaveBeenCalled()
    })

    it("sends the browser back to the app with the code on Allow", async () => {
        const auth = admin(claude)
        const leave = vi.fn()
        render(<ConnectPage request="sealed" auth={auth} leave={leave} onDismiss={vi.fn()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Allow"}))

        await vi.waitFor(() => expect(leave).toHaveBeenCalledWith("https://claude.ai/api/mcp/auth_callback?code=the-code"))
        expect(auth.approved).toEqual(["sealed"])
    })

    it("sends the browser back with a refusal on Deny, and approves nothing", async () => {
        const auth = admin(claude)
        const leave = vi.fn()
        render(<ConnectPage request="sealed" auth={auth} leave={leave} onDismiss={vi.fn()}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Deny"}))

        expect(leave).toHaveBeenCalledWith("https://claude.ai/api/mcp/auth_callback?error=access_denied")
        expect(auth.approved).toEqual([])
    })

    it("says a request expired, and lets the operator go to the backoffice instead", async () => {
        const auth = admin(() => Promise.reject(new ConnectError("ten minutes ago", Code.FailedPrecondition)))
        const dismiss = vi.fn()
        render(<ConnectPage request="sealed" auth={auth} leave={vi.fn()} onDismiss={dismiss}/>)

        expect(await screen.findByText("This request expired. Connect again from the app.")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Go to the backoffice"}))
        expect(dismiss).toHaveBeenCalled()
    })
})
