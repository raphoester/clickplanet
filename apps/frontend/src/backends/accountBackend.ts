import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {AuthService} from "../gen/grpc/auth/v1/auth_connect.ts"
import {
    LinkRefusal,
    LinkRefusalReason,
    Provider as WireProvider,
    SignInIntent,
} from "../gen/grpc/auth/v1/auth_pb.ts"
import {AccountBackend, AuthError, AuthFailure, Intent, Me, Provider} from "./account.ts"
import {retrying} from "./transport.ts"

const TO_WIRE: Record<Provider, WireProvider> = {
    google: WireProvider.GOOGLE,
    discord: WireProvider.DISCORD,
}

function providerOf(wire: WireProvider): Provider | undefined {
    return (Object.keys(TO_WIRE) as Provider[]).find((name) => TO_WIRE[name] === wire)
}

function providersOf(wire: WireProvider[]): Provider[] {
    return wire.map(providerOf).filter((p): p is Provider => p !== undefined)
}

const INTENTS: Record<Intent, SignInIntent> = {
    signIn: SignInIntent.SIGN_IN,
    link: SignInIntent.LINK,
}

const LINK_REFUSALS: Partial<Record<LinkRefusalReason, AuthFailure>> = {
    [LinkRefusalReason.IDENTITY_LINKED_ELSEWHERE]: "linkedElsewhere",
    [LinkRefusalReason.PROVIDER_ALREADY_LINKED]: "alreadyLinked",
}

const FAILURES: Partial<Record<Code, AuthFailure>> = {
    [Code.Unimplemented]: "off",
    [Code.InvalidArgument]: "notOffered",
    [Code.ResourceExhausted]: "tooManyTries",
    [Code.FailedPrecondition]: "startAgain",
    [Code.PermissionDenied]: "refused",
    [Code.Unauthenticated]: "notSignedIn",
}

async function mapped<T>(call: () => Promise<T>): Promise<T> {
    try {
        return await call()
    } catch (e) {
        const failure = e instanceof ConnectError ? refusalOf(e) : undefined
        throw new AuthError(failure ?? "failed", {cause: e})
    }
}

function refusalOf(e: ConnectError): AuthFailure | undefined {
    const link = e.findDetails(LinkRefusal)[0]
    return (link && LINK_REFUSALS[link.reason]) ?? FAILURES[e.code]
}

// Only reads are retried: a retried CompleteSignIn would spend a one-time code.
export class ConnectAccountBackend implements AccountBackend {
    constructor(private readonly client: PromiseClient<typeof AuthService>) {
    }

    public async signInOptions(): Promise<Provider[]> {
        try {
            const res = await retrying(() => this.client.getSignInOptions({}), "GetSignInOptions")
            return providersOf(res.providers)
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return []
            throw new AuthError("failed", {cause: e})
        }
    }

    public async me(): Promise<Me> {
        try {
            const res = await retrying(() => this.client.getMe({}), "GetMe")
            return {linked: providersOf(res.providers)}
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unauthenticated) return {linked: []}
            throw new AuthError("failed", {cause: e})
        }
    }

    public async startSignIn(provider: Provider, intent: Intent): Promise<string> {
        const res = await mapped(() => this.client.startSignIn({provider: TO_WIRE[provider], intent: INTENTS[intent]}))
        return res.authorizationUrl
    }

    public async completeSignIn(code: string, state: string): Promise<void> {
        await mapped(() => this.client.completeSignIn({code, state}))
    }

    public async signOut(): Promise<void> {
        await mapped(() => this.client.signOut({}))
    }

    public async signOutEverywhere(): Promise<void> {
        await mapped(() => this.client.signOutEverywhere({}))
    }

    public async deleteAccount(): Promise<void> {
        await mapped(() => this.client.deleteAccount({}))
    }
}
