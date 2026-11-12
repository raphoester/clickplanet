import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {ClipsService} from "../gen/grpc/clips/v1/appearances_connect.ts"
import {Appearance as AppearancePb, ClipLink as ClipLinkPb} from "../gen/grpc/clips/v1/appearances_pb.ts"
import {Appearance, ClipLink, ClipsBackend, ClipsNameBackend, Platform, ShownNamesBackend} from "./clips.ts"
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

// What GetShownNames takes at once.
const MOST_NAMES_ASKED = 100

export class ConnectClipsBackend implements ClipsBackend, ClipsNameBackend, ShownNamesBackend {
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

    public async nameShown(): Promise<boolean> {
        const headers = await this.identified()
        const res = await retrying(() => this.client.getAppearances({}, {headers}), "GetAppearances")
        return res.nameShown
    }

    public async setNameShown(shown: boolean): Promise<boolean> {
        const res = await this.client.setNameShown({nameShown: shown}, {headers: await this.identified()})
        return res.nameShown
    }

    public async shownNames(accountIds: readonly string[]): Promise<ReadonlyMap<string, string>> {
        const asked = [...new Set(accountIds)].filter((id) => id !== "").slice(0, MOST_NAMES_ASKED)
        if (asked.length === 0) return new Map()
        const res = await retrying(() => this.client.getShownNames({accountIds: asked}), "GetShownNames")
        return new Map(res.shownNames.map(({accountId, name}) => [accountId, name]))
    }

    private async identified(): Promise<Headers> {
        const token = await this.session.identity()
        if (!token) throw new Error("no identity token to answer for")
        return new Headers({[SESSION_HEADER]: token})
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
