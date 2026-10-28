import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {ClipsService} from "../gen/grpc/clips/v1/appearances_connect.ts"
import {Appearance as AppearancePb, ClipLink as ClipLinkPb} from "../gen/grpc/clips/v1/appearances_pb.ts"
import {Appearance, ClipLink, ClipsBackend, Platform} from "./clips.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, retrying} from "./transport.ts"

export function newClipsServiceClient(config: Config): PromiseClient<typeof ClipsService> {
    return createPromiseClient(ClipsService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectClipsBackend implements ClipsBackend {
    constructor(
        private readonly client: PromiseClient<typeof ClipsService>,
        private readonly session: SessionProvider,
    ) {
    }

    public async appearances(): Promise<Appearance[]> {
        const token = await this.session.identity()
        if (!token) return []

        const headers = new Headers({[SESSION_HEADER]: token})
        try {
            const res = await retrying(() => this.client.getAppearances({}, {headers}), "GetAppearances")
            return res.appearances.map(appearanceOf)
        } catch (e) {
            if (e instanceof ConnectError && [Code.Unimplemented, Code.NotFound, Code.Unauthenticated].includes(e.code)) return []
            throw e
        }
    }
}

function appearanceOf(appearance: AppearancePb): Appearance {
    return {
        postId: appearance.postId,
        title: appearance.title,
        scheduledAt: appearance.scheduledAt?.toDate().getTime() ?? 0,
        links: appearance.links.flatMap((link) => linkOf(link) ?? []),
    }
}

function linkOf(link: ClipLinkPb): ClipLink | undefined {
    const platform = platformOf(link.platform)
    return platform && {platform, url: link.url}
}

function platformOf(name: string): Platform | undefined {
    switch (name) {
        case "tiktok":
        case "youtube":
            return name
        default:
            return undefined
    }
}
