import {describe, expect, it, vi} from "vitest"
import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {Timestamp} from "@bufbuild/protobuf"
import {FlagService} from "../gen/grpc/flags/v1/flags_connect.ts"
import {
    Closure,
    ListMySubmissionsResponse,
    Picture,
    Pin,
    Rejection,
    RejectionReason,
    Submission,
    SubmissionState,
    SubmitFlagResponse,
} from "../gen/grpc/flags/v1/flags_pb.ts"
import {SubmissionRefusedError} from "./flags.ts"
import {ConnectFlagSubmissions, submissionOf} from "./flagSubmissionsBackend.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"

const session = (...tokens: string[]): SessionProvider => {
    const token = vi.fn(async () => tokens.length > 1 ? tokens.shift() : tokens[0])
    return {token, held: vi.fn(), identity: vi.fn(async () => tokens[0]), heldIdentity: vi.fn(), invalidate: vi.fn()}
}

const guestPage = (): SessionProvider => ({
    token: vi.fn(), held: vi.fn(), identity: vi.fn(async () => undefined), heldIdentity: vi.fn(), invalidate: vi.fn(),
})

const clientOf = (calls: Record<string, unknown>) => calls as unknown as PromiseClient<typeof FlagService>

const headersOf = (call: ReturnType<typeof vi.fn>, n = 0) => (call.mock.calls[n][1] as {headers: Headers}).headers

const waiting = () => new Submission({
    id: "s-1", name: "Bavaria", pin: new Pin({latitude: 48.1, longitude: 11.6}), state: SubmissionState.WAITING,
    submittedAt: Timestamp.fromDate(new Date(1_000)), decidedAt: Timestamp.fromDate(new Date(1_000)),
})

describe("submissionOf", () => {
    it("reads a rejection with its reason and the operator's note, and times in milliseconds", () => {
        const rejected = new Submission({
            ...waiting(), state: SubmissionState.REJECTED, decidedAt: Timestamp.fromDate(new Date(2_000)),
            rejection: new Rejection({reason: RejectionReason.HATEFUL, note: "No."}),
        })
        expect(submissionOf(rejected)).toEqual({
            id: "s-1", name: "Bavaria", pin: {latitude: 48.1, longitude: 11.6}, change: false, flag: "", state: "rejected",
            submittedAt: 1_000, decidedAt: 2_000, rejection: {reason: "hateful", note: "No."},
        })
    })

    it("reads the picture an approval makes public, and a change's flag", () => {
        const approved = submissionOf(new Submission({
            ...waiting(), state: SubmissionState.APPROVED, change: true, flag: "r-k3p9x2ma0b",
            picture: new Picture({url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512}),
        }))
        expect(approved?.change).toBe(true)
        expect(approved?.flag).toBe("r-k3p9x2ma0b")
        expect(approved?.picture?.url).toBe("https://media.clickplanet.lol/flags/pictures/ab.png")
    })

    it("drops a state this build does not know", () => {
        expect(submissionOf(new Submission({...waiting(), state: 9 as SubmissionState}))).toBeUndefined()
        expect(submissionOf(new Submission({...waiting(), state: SubmissionState.UNSPECIFIED}))).toBeUndefined()
    })
})

describe("ConnectFlagSubmissions", () => {
    it("uploads a picture: the link for its type and size with the click token, then the file put there", async () => {
        const createSubmissionUpload = vi.fn().mockResolvedValue({uploadId: "u-1", uploadUrl: "https://bucket/u-1"})
        const put = vi.fn(async () => undefined)
        const file = new Blob(["png"], {type: "image/png"})
        const onProgress = vi.fn()

        const id = await new ConnectFlagSubmissions(clientOf({createSubmissionUpload}), session("token-1"), put)
            .uploadPicture(file, {onProgress})

        expect(id).toBe("u-1")
        expect(createSubmissionUpload.mock.calls[0][0]).toEqual({contentType: "image/png", sizeBytes: 3n})
        expect(headersOf(createSubmissionUpload).get(SESSION_HEADER)).toBe("token-1")
        expect(put).toHaveBeenCalledWith("https://bucket/u-1", file, {onProgress})
    })

    it("submits a flag and answers the submission, a new flag with no flag named", async () => {
        const submitFlag = vi.fn().mockResolvedValue(new SubmitFlagResponse({submission: waiting()}))

        const submission = await new ConnectFlagSubmissions(clientOf({submitFlag}), session("token-1"))
            .submitFlag({name: "Bavaria", uploadId: "u-1", pin: {latitude: 48.1, longitude: 11.6}})

        expect(submitFlag.mock.calls[0][0]).toEqual({name: "Bavaria", uploadId: "u-1", pin: {latitude: 48.1, longitude: 11.6}, flag: ""})
        expect(submission.state).toBe("waiting")
    })

    it("mints again and sends once more when the token is refused, never a third time", async () => {
        const submitFlag = vi.fn()
            .mockRejectedValueOnce(new ConnectError("no token", Code.Unauthenticated))
            .mockResolvedValue(new SubmitFlagResponse({submission: waiting()}))
        const provider = session("stale", "fresh")

        await new ConnectFlagSubmissions(clientOf({submitFlag}), provider)
            .submitFlag({name: "Bavaria", uploadId: "u-1", pin: {latitude: 0, longitude: 0}})

        expect(provider.invalidate).toHaveBeenCalledTimes(1)
        expect(headersOf(submitFlag, 1).get(SESSION_HEADER)).toBe("fresh")

        const refused = vi.fn().mockRejectedValue(new ConnectError("no token", Code.Unauthenticated))
        await expect(new ConnectFlagSubmissions(clientOf({submitFlag: refused}), session("t"))
            .submitFlag({name: "Bavaria", uploadId: "u-1", pin: {latitude: 0, longitude: 0}}))
            .rejects.toMatchObject({refusal: "no-session"})
        expect(refused).toHaveBeenCalledTimes(2)
    })

    it("says each refusal as its reason: a guest's, a flag another account founded, one that waits", async () => {
        const refusalOf = async (code: Code, flag?: string) => {
            const submitFlag = vi.fn().mockRejectedValue(new ConnectError("refused", code))
            const sent = new ConnectFlagSubmissions(clientOf({submitFlag}), session("t"))
                .submitFlag({name: "Bavaria", uploadId: "u-1", pin: {latitude: 0, longitude: 0}, flag})
            return sent.catch((e: unknown) => e instanceof SubmissionRefusedError ? e.refusal : e)
        }

        expect(await refusalOf(Code.PermissionDenied)).toBe("guest")
        expect(await refusalOf(Code.PermissionDenied, "r-k3p9x2ma0b")).toBe("not-yours")
        expect(await refusalOf(Code.AlreadyExists)).toBe("waiting")
        expect(await refusalOf(Code.InvalidArgument)).toBe("invalid")
        expect(await refusalOf(Code.FailedPrecondition)).toBe("off")
        expect(await refusalOf(Code.ResourceExhausted)).toBe("too-many")
        expect(await refusalOf(Code.Internal)).toBeInstanceOf(ConnectError)
    })

    it("reads the account's submissions with the token that names it, and none with no token", async () => {
        const listMySubmissions = vi.fn().mockResolvedValue(new ListMySubmissionsResponse({submissions: [waiting()]}))

        const mine = await new ConnectFlagSubmissions(clientOf({listMySubmissions}), session("token-1")).mySubmissions()
        expect(mine.map(({id}) => id)).toEqual(["s-1"])
        expect(headersOf(listMySubmissions).get(SESSION_HEADER)).toBe("token-1")

        const none = vi.fn()
        expect(await new ConnectFlagSubmissions(clientOf({listMySubmissions: none}), guestPage()).mySubmissions()).toEqual([])
        expect(none).not.toHaveBeenCalled()
    })

    it("puts the note of a flag's close on the submission that opened it, and on no other", async () => {
        const opened = new Submission({...waiting(), id: "s-1", flag: "r-k3p9x2ma0b", state: SubmissionState.OPENED})
        const change = new Submission({...waiting(), id: "s-2", flag: "r-k3p9x2ma0b", change: true, state: SubmissionState.APPROVED})
        const listMySubmissions = vi.fn().mockResolvedValue(new ListMySubmissionsResponse({
            submissions: [change, opened],
            closures: [new Closure({flag: "r-k3p9x2ma0b", note: "A club's crest."})],
        }))

        const mine = await new ConnectFlagSubmissions(clientOf({listMySubmissions}), session("t")).mySubmissions()

        expect(mine.map(({id, closeNote}) => [id, closeNote])).toEqual([["s-2", undefined], ["s-1", "A club's crest."]])
    })

    it("reads none from a server without submissions", async () => {
        const listMySubmissions = vi.fn().mockRejectedValue(new ConnectError("no", Code.Unimplemented))
        expect(await new ConnectFlagSubmissions(clientOf({listMySubmissions}), session("t")).mySubmissions()).toEqual([])
    })
})
