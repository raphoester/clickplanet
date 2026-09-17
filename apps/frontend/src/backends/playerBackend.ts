import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {PlayerService} from "../gen/grpc/player/v1/player_connect.ts"
import {Player as PlayerPb, Profile as ProfilePb, RosterEntry as RosterEntryPb} from "../gen/grpc/player/v1/player_pb.ts"
import {
    PlayerBackend,
    PlayerError,
    PlayerFailure,
    PlayerInfo,
    PlayerInfoBackend,
    Presence,
    PresenceBackend,
    Profile,
    RosterEntry,
    RosterUnavailableError,
} from "./player.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, retrying} from "./transport.ts"

/**
 * No cookie: the account is named by the click token in a header, not by
 * `cp_sid`. `useHttpGet` sends `GetRoster` and `GetPlayer`, the calls marked
 * side-effect free, as GETs a proxy can cache; every other call stays a POST.
 */
export function newPlayerServiceClient(config: Config): PromiseClient<typeof PlayerService> {
    return createPromiseClient(PlayerService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

const FAILURES: Partial<Record<Code, PlayerFailure>> = {
    [Code.InvalidArgument]: "invalid",
    [Code.AlreadyExists]: "taken",
    [Code.PermissionDenied]: "guest",
    [Code.Unauthenticated]: "notSignedIn",
}

/**
 * `player.v1.PlayerService` behind `PlayerBackend`.
 *
 * **A call refused for its session is sent once more with a fresh one**, as
 * `PlanetBackend` does for a click: the token may name the account the browser
 * was on before a sign-in. Only the read is retried while the server cannot be
 * reached — `SetName` is a write, like every other one here.
 */
export class ConnectPlayerBackend implements PlayerBackend, PresenceBackend, PlayerInfoBackend {
    constructor(
        private readonly client: PromiseClient<typeof PlayerService>,
        private readonly session: SessionProvider,
    ) {
    }

    public async profile(): Promise<Profile> {
        const res = await this.authenticated((headers) =>
            retrying(() => this.client.getProfile({}, {headers}), "GetProfile"))
        return profileOf(res.profile)
    }

    public async setName(name: string): Promise<Profile> {
        const res = await this.authenticated((headers) => this.client.setName({name}, {headers}))
        return profileOf(res.profile)
    }

    public heldSession(): string | undefined {
        return this.session.held()
    }

    /**
     * With the token already held, never a fresh one: a mint is a Turnstile
     * check, and presence is not worth one. So a refusal for the session is not
     * retried the way `authenticated` retries — the token is dropped, since the
     * server said it is no good, and the next click mints the one the next
     * announce goes out with. Not retried while the server cannot be reached
     * either: the schedule sends another in 30s.
     */
    public async announce(presence: Presence): Promise<boolean> {
        const token = this.session.held()
        if (!token) return false

        const headers = new Headers({[SESSION_HEADER]: token})
        try {
            await this.client.announce({countryId: presence.countryCode, guestName: presence.guestName}, {headers})
            return true
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unauthenticated) this.session.invalidate()
            const failure = e instanceof ConnectError ? FAILURES[e.code] : undefined
            throw new PlayerError(failure ?? "failed", {cause: e})
        }
    }

    /**
     * No token and no header: a custom header would cost a preflight, and a
     * read that names its caller is one no proxy shares. A 404 is a server
     * without the call — connect-web reads it as `unimplemented`.
     */
    public async roster(): Promise<RosterEntry[]> {
        try {
            const res = await retrying(() => this.client.getRoster({}), "GetRoster")
            return res.entries.map(rosterEntryOf)
        } catch (e) {
            if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) {
                throw new RosterUnavailableError({cause: e})
            }
            throw e
        }
    }

    /**
     * No token, like the roster: anybody may read a player, and a proxy may
     * serve one answer to everyone who opens it.
     */
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
    return {name: entry.name, tag: entry.tag, countryCode: entry.countryId, guest: entry.guest}
}

function playerInfoOf(player: PlayerPb | undefined): PlayerInfo {
    const createdAt = Number(player?.createdAtUnixMs ?? 0)
    return {
        name: player?.name ?? "",
        tilesTaken: Number(player?.stats?.tilesTaken ?? 0),
        streakCurrent: player?.stats?.streakCurrent ?? 0,
        streakBest: player?.stats?.streakBest ?? 0,
        createdAt: createdAt > 0 ? createdAt : undefined,
    }
}
