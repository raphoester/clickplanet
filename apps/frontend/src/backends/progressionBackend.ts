import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {ProgressionService} from "../gen/grpc/progression/v1/progression_connect.ts"
import {
    CountryTiles as CountryTilesPb,
    Progress as ProgressPb,
    ProgressionEvent,
    Track as TrackPb,
} from "../gen/grpc/progression/v1/progression_pb.ts"
import {AccountBadge, BadgesBackend} from "./badges.ts"
import {
    CountryTiles,
    Fronts,
    PlayerTitle,
    Progress,
    ProgressionBackend,
    ProgressReader,
    TitleDashboard,
    TitlesListener,
    TitleTrack,
} from "./progression.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {titleOf} from "./title.ts"
import {Config, NO_TIMEOUT, openStream, retrying} from "./transport.ts"

export function newProgressionServiceClient(config: Config): PromiseClient<typeof ProgressionService> {
    return createPromiseClient(ProgressionService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectProgressionBackend implements ProgressionBackend, ProgressReader, BadgesBackend, TitlesListener {
    constructor(
        private readonly client: PromiseClient<typeof ProgressionService>,
        private readonly session: SessionProvider,
    ) {
    }

    public async titles(): Promise<TitleDashboard> {
        const res = await this.authenticated((headers) =>
            retrying(() => this.client.getTitles({}, {headers}), "GetTitles"))
        return {
            worn: titleOf(res.worn),
            wearable: res.wearable.flatMap((title) => titleOf(title) ?? []),
            tracks: res.tracks.map(trackOf),
        }
    }

    public async wearTitle(id: string): Promise<PlayerTitle | undefined> {
        const res = await this.authenticated((headers) => this.client.wearTitle({titleId: id}, {headers}))
        return titleOf(res.worn)
    }

    public async fronts(): Promise<Fronts> {
        const res = await this.authenticated((headers) =>
            retrying(() => this.client.getFronts({}, {headers}), "GetFronts"))
        return {playsFor: res.playsFor.map(countryTilesOf), playsAgainst: res.playsAgainst.map(countryTilesOf)}
    }

    public async progress(accountId: string): Promise<Progress | undefined> {
        try {
            const res = await retrying(() => this.client.getProgress({accountId}), "GetProgress")
            return progressOf(res.progress)
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.NotFound) return undefined
            throw e
        }
    }

    // No retry here: the badges store reads them again later.
    public async badges(accountIds: readonly string[]): Promise<AccountBadge[]> {
        const res = await this.client.getBadges({accountIds: [...accountIds]})
        return res.badges.map((badge) => ({
            accountId: badge.accountId,
            streak: badge.streak,
            wornTitle: titleOf(badge.wornTitle),
        }))
    }

    public heldIdentity(): string | undefined {
        return this.session.heldIdentity()
    }

    public listenForTitles(onEarned: (title: PlayerTitle) => void): () => void {
        const client = this.client
        const session = this.session
        return openStream(
            (signal) => {
                const headers = new Headers()
                const token = session.heldIdentity()
                if (token) headers.set(SESSION_HEADER, token)
                return client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT, headers})
            },
            (event) => {
                const title = titleEarnedOf(event)
                if (title) onEarned(title)
            },
            "titles",
            {stopsOn: (e) => e instanceof ConnectError && e.code === Code.Unauthenticated},
        )
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
        const token = await this.session.token()

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }
}

function titleEarnedOf(event: ProgressionEvent): PlayerTitle | undefined {
    return event.event.case === "titleEarned" ? titleOf(event.event.value.title) : undefined
}

function progressOf(progress: ProgressPb | undefined): Progress {
    return {
        tilesTaken: Number(progress?.stats?.tilesTaken ?? 0),
        streakCurrent: progress?.stats?.streakCurrent ?? 0,
        streakBest: progress?.stats?.streakBest ?? 0,
        titles: (progress?.titles ?? []).flatMap((title) => titleOf(title) ?? []),
        wornTitle: titleOf(progress?.wornTitle),
        playsFor: (progress?.playsFor ?? []).map(countryTilesOf),
        playsAgainst: (progress?.playsAgainst ?? []).map(countryTilesOf),
    }
}

function countryTilesOf(country: CountryTilesPb): CountryTiles {
    return {countryCode: country.countryId, tiles: Number(country.tiles)}
}

function trackOf(track: TrackPb): TitleTrack {
    return {
        id: track.id,
        name: track.name,
        progress: Number(track.progress),
        steps: track.steps.flatMap((step) => {
            const title = titleOf(step.title)
            return title ? [{title, threshold: Number(step.threshold), earned: step.earned}] : []
        }),
    }
}
