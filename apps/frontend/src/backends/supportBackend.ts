import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {SupportService} from "../gen/grpc/support/v1/support_connect.ts";
import {ReportReason as ReportReasonPb} from "../gen/grpc/support/v1/support_pb.ts";
import {SESSION_HEADER, SessionProvider} from "./session.ts";
import {
    BugContext,
    BugReporter,
    FeatureSuggester,
    PlayerReport,
    PlayerReporter,
    ReportReason,
    SupportNoSessionError,
    SupportRateLimitedError,
    SupportRejectedError,
    SupportReportedAlreadyError,
    SupportUnknownPlayerError,
} from "./support.ts";
import {Config} from "./transport.ts";

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

export class ConnectSupportBackend implements FeatureSuggester, BugReporter, PlayerReporter {
    constructor(
        private client: PromiseClient<typeof SupportService>,
        private readonly session: SessionProvider,
    ) {
    }

    public suggestFeature(text: string): Promise<void> {
        return this.send((headers) => this.client.suggestFeature({text}, {headers}))
    }

    public reportBug(text: string, context: BugContext): Promise<void> {
        return this.send((headers) => this.client.reportBug({text, context}, {headers}))
    }

    public reportPlayer(report: PlayerReport): Promise<void> {
        return this.send((headers) => this.client.reportPlayer({
            name: report.name,
            reason: REASONS[report.reason],
            text: report.text,
            messageId: report.messageId ?? "",
        }, {headers}))
    }

    // Not retried: a call that landed and lost its answer would be kept twice.
    private async send(call: (headers: Headers) => Promise<unknown>): Promise<void> {
        try {
            await this.authenticated(call)
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

function translate(e: unknown): unknown {
    if (!(e instanceof ConnectError)) return e

    switch (e.code) {
        case Code.ResourceExhausted:
            return new SupportRateLimitedError({cause: e})
        case Code.InvalidArgument:
            return new SupportRejectedError({cause: e})
        case Code.NotFound:
            return new SupportUnknownPlayerError({cause: e})
        case Code.AlreadyExists:
            return new SupportReportedAlreadyError({cause: e})
        case Code.Unauthenticated:
            return new SupportNoSessionError({cause: e})
        default:
            return e
    }
}
