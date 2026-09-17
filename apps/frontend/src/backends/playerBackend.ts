import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {PlayerService} from "../gen/grpc/player/v1/player_connect.ts"
import {Profile as ProfilePb} from "../gen/grpc/player/v1/player_pb.ts"
import {PlayerBackend, PlayerError, PlayerFailure, Profile} from "./player.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, retrying} from "./transport.ts"

/** No cookie: the account is named by the click token in a header, not by `cp_sid`. */
export function newPlayerServiceClient(config: Config): PromiseClient<typeof PlayerService> {
    return createPromiseClient(PlayerService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
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
export class ConnectPlayerBackend implements PlayerBackend {
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
