import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {FlagService} from "../gen/grpc/flags/v1/flags_connect.ts"
import {
    RejectionReason as RejectionReasonPb,
    Submission as SubmissionPb,
    SubmissionState as SubmissionStatePb,
} from "../gen/grpc/flags/v1/flags_pb.ts"
import {MyFlags, NO_FLAGS, RejectionReason, Submission, SubmissionState} from "../domain/flagSubmissions.ts"
import {AppealRefusedError, FlagSubmitter, SubmissionRefusal, SubmissionRefusedError, SubmittedFlag} from "./flags.ts"
import {anthemMessageOf, anthemPart, pictureOf, pinOf} from "./flagsBackend.ts"
import {putFile, PutFile} from "./putFile.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {UploadOptions} from "./support.ts"
import {retrying} from "./transport.ts"

const STATES: Record<SubmissionStatePb, SubmissionState | undefined> = {
    [SubmissionStatePb.UNSPECIFIED]: undefined,
    [SubmissionStatePb.WAITING]: "waiting",
    [SubmissionStatePb.APPROVED]: "approved",
    [SubmissionStatePb.REJECTED]: "rejected",
    [SubmissionStatePb.OPENED]: "opened",
}

const REASONS: Record<RejectionReasonPb, RejectionReason> = {
    [RejectionReasonPb.UNSPECIFIED]: "other",
    [RejectionReasonPb.NOT_A_PLACE]: "not-a-place",
    [RejectionReasonPb.EXISTS]: "exists",
    [RejectionReasonPb.UNREADABLE]: "unreadable",
    [RejectionReasonPb.HATEFUL]: "hateful",
    [RejectionReasonPb.OTHER]: "other",
}

// A state this build does not know is dropped: a new enum value must not break an old tab.
export function submissionOf(submission: SubmissionPb): Submission | undefined {
    const state = STATES[submission.state]
    if (state === undefined) return undefined
    return {
        id: submission.id,
        name: submission.name,
        ...submission.pin ? {pin: pinOf(submission.pin)} : {},
        flag: submission.flag,
        state,
        submittedAt: submission.submittedAt?.toDate().getTime() ?? 0,
        decidedAt: submission.decidedAt?.toDate().getTime() ?? 0,
        ...state === "rejected" && submission.rejection
            ? {rejection: {reason: REASONS[submission.rejection.reason] ?? "other", note: submission.rejection.note}}
            : {},
        ...submission.picture && submission.picture.url !== "" ? {picture: pictureOf(submission.picture)} : {},
        paid: submission.paid,
        ...submission.rejectedByJudge ? {rejectedByJudge: true} : {},
        ...submission.appeal ? {appealed: true} : {},
        ...anthemPart(submission.anthem),
    }
}

// The flags module's submissions, over the flags client: the two sends with the click token, the read with any token
// that names the account.
export class ConnectFlagSubmissions implements FlagSubmitter {
    constructor(
        private readonly client: PromiseClient<typeof FlagService>,
        private readonly session: SessionProvider,
        private readonly put: PutFile = putFile,
    ) {
    }

    public async uploadPicture(file: Blob, options?: UploadOptions): Promise<string> {
        const upload = await this.send(false, (headers) =>
            this.client.createSubmissionUpload({contentType: file.type, sizeBytes: BigInt(file.size)}, {headers}))
        await this.put(upload.uploadUrl, file, options)
        return upload.uploadId
    }

    // Not retried: a call that landed and lost its answer would wait twice, and the second would be refused.
    public async submitFlag({name, uploadId, pin, replaces, anthem}: SubmittedFlag): Promise<Submission> {
        const res = await this.send(replaces !== undefined, (headers) =>
            this.client.submitFlag({name, uploadId, pin, replaces: replaces ?? "", anthem: anthemMessageOf(anthem)},
                {headers}))
        const submission = res.submission && submissionOf(res.submission)
        if (!submission) throw new Error("the flags module answered no submission it can read")
        return submission
    }

    // Not retried: a call that landed and lost its answer would be refused the second time.
    public async appealSubmission(id: string): Promise<Submission> {
        try {
            const res = await this.authenticated((headers) => this.client.appealSubmission({submissionId: id}, {headers}))
            const submission = res.submission && submissionOf(res.submission)
            if (!submission) throw new Error("the flags module answered no submission it can read")
            return submission
        } catch (e) {
            if (!(e instanceof ConnectError)) throw e
            if (e.code === Code.AlreadyExists) throw new AppealRefusedError("waiting", {cause: e})
            if ([Code.FailedPrecondition, Code.NotFound, Code.PermissionDenied].includes(e.code)) {
                throw new AppealRefusedError("refused", {cause: e})
            }
            throw e
        }
    }

    // Retried: the same anthem set twice is set once. The flags' stream tells the page the flag with it.
    public async setAnthem(flag: string, anthem: string | undefined): Promise<void> {
        await this.send(false, (headers) => retrying(
            () => this.client.setAnthem({flag, anthem: anthemMessageOf(anthem)}, {headers}), "SetAnthem"))
    }

    // A removal's note goes on the flag held it is for.
    public async myFlags(): Promise<MyFlags> {
        const token = await this.session.identity()
        if (!token) return NO_FLAGS
        const headers = new Headers({[SESSION_HEADER]: token})
        try {
            const res = await retrying(() => this.client.listMySubmissions({}, {headers}), "ListMySubmissions")
            const notes = new Map(res.closures.map(({flag, note}) => [flag, note]))
            return {
                submissions: res.submissions.flatMap((submission) => submissionOf(submission) ?? []),
                held: res.held.map((code) => {
                    const removalNote = notes.get(code)
                    return removalNote === undefined ? {code} : {code, removalNote}
                }),
            }
        } catch (e) {
            if (e instanceof ConnectError && [Code.Unimplemented, Code.Unauthenticated].includes(e.code)) return NO_FLAGS
            throw e
        }
    }

    private async send<T>(replacing: boolean, call: (headers: Headers) => Promise<T>): Promise<T> {
        try {
            return await this.authenticated(call)
        } catch (e) {
            throw refusalOf(e, replacing)
        }
    }

    private async authenticated<T>(call: (headers: Headers) => Promise<T>): Promise<T> {
        try {
            return await call(await this.headers())
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw e

            this.session.invalidate()
            return await call(await this.headers())
        }
    }

    private async headers(): Promise<Headers> {
        let token: string | undefined
        try {
            token = await this.session.token()
        } catch (e) {
            throw new SubmissionRefusedError("no-session", {cause: e})
        }

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }
}

const REFUSALS: Partial<Record<Code, SubmissionRefusal>> = {
    [Code.Unauthenticated]: "no-session",
    [Code.PermissionDenied]: "guest",
    [Code.AlreadyExists]: "waiting",
    [Code.InvalidArgument]: "invalid",
    [Code.FailedPrecondition]: "off",
    [Code.ResourceExhausted]: "too-many",
}

// FailedPrecondition for a replacement is the operator's decision on the one it replaced, more likely than pictures off.
function refusalOf(e: unknown, replacing: boolean): unknown {
    if (!(e instanceof ConnectError)) return e

    const refusal = e.code === Code.FailedPrecondition && replacing ? "decided" : REFUSALS[e.code]
    return refusal ? new SubmissionRefusedError(refusal, {cause: e}) : e
}
