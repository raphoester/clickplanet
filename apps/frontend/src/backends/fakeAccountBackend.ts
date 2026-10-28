import {AccountBackend, AuthError, Me, Provider} from "./account.ts"
import {FAKE_YOU} from "./fakeAccounts.ts"
import {ColoredProfile, Fronts, NameColor, PlayerBackend, PlayerError, PlayerTitle, Profile, TitleDashboard} from "./player.ts"

// Offers no provider: the console's signIn() links it, as a finished sign-in would.
export class FakeAccountBackend implements AccountBackend, PlayerBackend {
    private linked: Provider[] = []
    private name = ""
    private color = NameColor.UNSPECIFIED

    public signIn(name: string): void {
        this.linked = ["google"]
        this.name = name
    }

    public async signInOptions(): Promise<Provider[]> {
        return []
    }

    public async me(): Promise<Me> {
        return {linked: [...this.linked]}
    }

    public async startSignIn(): Promise<string> {
        throw new AuthError("off")
    }

    public async completeSignIn(): Promise<void> {
        throw new AuthError("off")
    }

    public async startEmailSignIn(): Promise<void> {
        throw new AuthError("off")
    }

    public async completeEmailSignIn(): Promise<void> {
        throw new AuthError("off")
    }

    public async signOut(): Promise<void> {
        this.forget()
    }

    public async signOutEverywhere(): Promise<void> {
        this.forget()
    }

    public async deleteAccount(): Promise<void> {
        this.forget()
    }

    public async profile(): Promise<ColoredProfile> {
        return {accountId: FAKE_YOU, name: this.name, color: this.color}
    }

    public async setName(name: string): Promise<Profile> {
        this.requireSignedIn()
        this.name = name
        return {accountId: FAKE_YOU, name}
    }

    public async setColor(color: NameColor): Promise<NameColor> {
        this.requireSignedIn()
        this.color = color
        return color
    }

    public async titles(): Promise<TitleDashboard> {
        return {wearable: [], tracks: []}
    }

    public async wearTitle(): Promise<PlayerTitle | undefined> {
        return undefined
    }

    public async fronts(): Promise<Fronts> {
        return {playsFor: [], playsAgainst: []}
    }

    private requireSignedIn() {
        if (this.linked.length === 0) throw new PlayerError("guest")
    }

    private forget() {
        this.linked = []
        this.name = ""
        this.color = NameColor.UNSPECIFIED
    }
}
