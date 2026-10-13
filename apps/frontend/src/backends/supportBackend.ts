import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {SupportService} from "../gen/grpc/support/v1/support_connect.ts";
import {ReportReason as ReportReasonPb} from "../gen/grpc/support/v1/support_pb.ts";
import {putFile, PutFile} from "./putFile.ts";
import {SESSION_HEADER, SessionProvider} from "./session.ts";
import {
    AttachmentRules,
    AttachmentUpload,
    NO_ATTACHMENTS,
    PageContext,
    PlayerReport,
    ReportReason,
    SupportBackend,
    SupportNoSessionError,
    SupportRateLimitedError,
    SupportRejectedError,
    SupportReportedAlreadyError,
    UploadOptions,
} from "./support.ts";
import {Config, retrying} from "./transport.ts";

export function newSupportServiceClient(config: Config): PromiseClient<typeof SupportService> {
    return createPromiseClient(SupportService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

const REASONS: Record<ReportReason, ReportReasonPb> = {
    "cheating": ReportReasonPb.CHEATING,
    "chat-abuse": ReportReasonPb.CHAT_ABUSE,
    "bad-name": ReportReasonPb.BAD_NAME,
    "other": ReportReasonPb.OTHER,
}

export class ConnectSupportBackend implements SupportBackend {
    constructor(
        private client: PromiseClient<typeof SupportService>,
        private readonly session: SessionProvider,
        private readonly put: PutFile = putFile,
    ) {
    }

    public async suggestFeature(text: string, context: PageContext, attachments: Blob[] = []): Promise<AttachmentUpload[]> {
        return uploadsOf(await this.send((headers) =>
            this.client.suggestFeature({text, context, attachments: declared(attachments)}, {headers})))
    }

    public async reportBug(text: string, context: PageContext, attachments: Blob[] = []): Promise<AttachmentUpload[]> {
        return uploadsOf(await this.send((headers) =>
            this.client.reportBug({text, context, attachments: declared(attachments)}, {headers})))
    }

    // A server from before attachments has no such procedure: it takes none.
    public async attachmentRules(): Promise<AttachmentRules> {
        try {
            const rules = await retrying(() => this.client.getAttachmentRules({}), "attachment rules")
            return {
                maxPerNote: rules.maxPerNote,
                maxBytes: Number(rules.maxBytes),
                contentTypes: rules.contentTypes,
            }
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return NO_ATTACHMENTS
            throw e
        }
    }

    public uploadAttachment(upload: AttachmentUpload, file: Blob, options?: UploadOptions): Promise<void> {
        return this.put(upload.url, file, options)
    }

    public async reportPlayer(report: PlayerReport): Promise<void> {
        await this.send((headers) => this.client.reportPlayer({
            accountId: report.accountId,
            reason: REASONS[report.reason],
            text: report.text,
            messageId: report.messageId ?? "",
        }, {headers}))
    }

    // Not retried: a call that landed and lost its answer would be kept twice.
    private async send<T>(call: (headers: Headers) => Promise<T>): Promise<T> {
        try {
            return await this.authenticated(call)
        } catch (e) {
            throw translate(e)
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
            throw new SupportNoSessionError({cause: e})
        }

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }
}

function declared(attachments: Blob[]) {
    return attachments.map((file) => ({contentType: file.type, sizeBytes: BigInt(file.size)}))
}

function uploadsOf(sent: {uploads: {url: string}[]}): AttachmentUpload[] {
    return sent.uploads.map(({url}) => ({url}))
}

function translate(e: unknown): unknown {
    if (!(e instanceof ConnectError)) return e

    switch (e.code) {
        case Code.ResourceExhausted:
            return new SupportRateLimitedError({cause: e})
        case Code.InvalidArgument:
            return new SupportRejectedError({cause: e})
        case Code.AlreadyExists:
            return new SupportReportedAlreadyError({cause: e})
        case Code.Unauthenticated:
            return new SupportNoSessionError({cause: e})
        default:
            return e
    }
}
