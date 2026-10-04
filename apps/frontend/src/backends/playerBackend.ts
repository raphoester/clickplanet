import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {PlayerService} from "../gen/grpc/player/v1/player_connect.ts"
import {
    Player as PlayerPb,
    PlayerEvent as PlayerEventPb,
    Profile as ProfilePb,
    RosterEntry as RosterEntryPb,
    Title as TitlePb,
    Track as TrackPb,
} from "../gen/grpc/player/v1/player_pb.ts"
import {
    ColoredProfile,
    NameColor,
    PlayerBackend,
    PlayerError,
    PlayerFailure,
    PlayerInfo,
    PlayerInfoBackend,
    PlayerTitle,
    Presence,
    PresenceBackend,
    Profile,
    RosterEntry,
    RosterEvent,
    TitleDashboard,
    TitleTrack,
} from "./player.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, NO_TIMEOUT, openStream, retrying} from "./transport.ts"

export function newPlayerServiceClient(config: Config): PromiseClient<typeof PlayerService> {
    return createPromiseClient(PlayerService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export function newKeepalivePlayerServiceClient(config: Config): PromiseClient<typeof PlayerService> {
    return createPromiseClient(PlayerService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        fetch: (input, init) => fetch(input, {...init, keepalive: true}),
    }))
}

const FAILURES: Partial<Record<Code, PlayerFailure>> = {
    [Code.InvalidArgument]: "invalid",
    [Code.AlreadyExists]: "taken",
    [Code.PermissionDenied]: "guest",
    [Code.Unauthenticated]: "notSignedIn",
    [Code.FailedPrecondition]: "unnamed",
}

export class ConnectPlayerBackend implements PlayerBackend, PresenceBackend, PlayerInfoBackend {
    constructor(
        private readonly client: PromiseClient<typeof PlayerService>,
        private readonly session: SessionProvider,
        private readonly keepalive: PromiseClient<typeof PlayerService>,
    ) {
    }

    public async profile(): Promise<ColoredProfile> {
        const res = await this.authenticated((headers) =>
            retrying(() => this.client.getProfile({}, {headers}), "GetProfile"))
        return {...profileOf(res.profile), color: res.color}
    }

    public async setName(name: string): Promise<Profile> {
        const res = await this.authenticated((headers) => this.client.setName({name}, {headers}))
        return profileOf(res.profile)
    }

    public async setColor(color: NameColor): Promise<NameColor> {
        const res = await this.authenticated((headers) => this.client.setColor({color}, {headers}))
        return res.color
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

    public heldSession(): string | undefined {
        return this.session.held()
    }

    public async announce(presence: Presence): Promise<boolean> {
        const token = this.session.held()
        if (!token) return false

        const headers = new Headers({[SESSION_HEADER]: token})
        try {
            await this.client.announce({countryId: presence.countryCode}, {headers})
            return true
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unauthenticated) this.session.invalidate()
            const failure = e instanceof ConnectError ? FAILURES[e.code] : undefined
            throw new PlayerError(failure ?? "failed", {cause: e})
        }
    }

    public leave(): void {
        const token = this.session.held()
        if (!token) return

        this.keepalive.leave({}, {headers: new Headers({[SESSION_HEADER]: token})})
            .catch((e) => console.error("Leave failed", e))
    }

    public listenForRoster(
        onEvent: (event: RosterEvent) => void,
        onUnavailable: () => void,
        onTitleEarned: (title: PlayerTitle) => void,
    ): () => void {
        const client = this.client
        const session = this.session
        let unavailable = false
        let stop = () => {}
        stop = openStream(
            async function* (signal) {
                const token = session.held()
                const headers = new Headers()
                if (token) headers.set(SESSION_HEADER, token)
                try {
                    yield* client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT, headers})
                } catch (e) {
                    if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) {
                        unavailable = true
                        onUnavailable()
                        stop()
                        return
                    }
                    throw e
                }
            },
            (event) => {
                if (unavailable) return
                if (event.event.case === "titleEarned") {
                    const title = titleOf(event.event.value.title)
                    if (title) onTitleEarned(title)
                    return
                }
                const rosterEvent = rosterEventOf(event)
                if (rosterEvent) onEvent(rosterEvent)
            },
            "roster",
        )
        return () => stop()
    }

    public async playerInfo(name: string): Promise<PlayerInfo | undefined> {
        try {
            const res = await retrying(() => this.client.getPlayer({name}), "GetPlayer")
            return playerInfoOf(res.player)
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.NotFound) return undefined
            throw e
        }
    }

    private async authenticated<T>(call: (headers: Headers) => Promise<T>): Promise<T> {
        try {
            try {
                return await call(await this.headers())
            } catch (e) {
                if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw e

                this.session.invalidate()
                return await call(await this.headers())
            }
        } catch (e) {
            const failure = e instanceof ConnectError ? FAILURES[e.code] : undefined
            throw new PlayerError(failure ?? "failed", {cause: e})
        }
    }

    private async headers(): Promise<Headers> {
        const token = await this.session.token()

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }
}

function profileOf(profile: ProfilePb | undefined): Profile {
    return {accountId: profile?.accountId ?? "", name: profile?.name ?? ""}
}

function rosterEntryOf(entry: RosterEntryPb): RosterEntry {
    return {
        key: entry.key,
        name: entry.name,
        countryCode: entry.countryId,
        guest: entry.guest,
        admin: entry.admin,
        color: entry.color,
        streak: entry.streak,
    }
}

function rosterEventOf(event: PlayerEventPb): RosterEvent | undefined {
    switch (event.event.case) {
        case "roster":
            return {kind: "roster", entries: event.event.value.entries.map(rosterEntryOf)}
        case "entry":
            return {kind: "entry", entry: rosterEntryOf(event.event.value)}
        case "left":
            return {kind: "left", key: event.event.value.key}
        default:
            return undefined
    }
}

function playerInfoOf(player: PlayerPb | undefined): PlayerInfo {
    const createdAt = Number(player?.createdAtUnixMs ?? 0)
    return {
        name: player?.name ?? "",
        tilesTaken: Number(player?.stats?.tilesTaken ?? 0),
        streakCurrent: player?.stats?.streakCurrent ?? 0,
        streakBest: player?.stats?.streakBest ?? 0,
        createdAt: createdAt > 0 ? createdAt : undefined,
        admin: player?.admin ?? false,
        color: player?.color ?? NameColor.UNSPECIFIED,
        titles: (player?.titles ?? []).flatMap((title) => titleOf(title) ?? []),
        wornTitle: titleOf(player?.wornTitle),
    }
}

function titleOf(title: TitlePb | undefined): PlayerTitle | undefined {
    if (!title || !title.id) return undefined

    const rank = title.rank
    return {
        id: title.id,
        name: title.name,
        rank: rank && rank.count > 0
            ? {trackId: rank.trackId, trackName: rank.trackName, number: rank.number, count: rank.count}
            : undefined,
    }
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
