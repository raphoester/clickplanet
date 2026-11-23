import {AccountBackend, AuthError, Me, OAuthProvider, Provider, SignInOutcome} from "./account.ts"
import {CALLBACK_PATH} from "../domain/signInCallback.ts"
import {FAKE_YOU} from "./fakeAccounts.ts"
import {FAKE_MISSION_RATES} from "./fakeMissionsBackend.ts"
import {Nameplate} from "./nameplates.ts"
import {ColoredProfile, NameColor, PlayerBackend, PlayerError, Profile} from "./player.ts"
import {Fronts, LevelRead, LevelReader, PlayerLevel, PlayerTitle, ProgressionBackend, TitleDashboard, XpRates} from "./progression.ts"

// Offers no provider until offerSignIn(). completeSignIn takes the code for the name to sign in with: the console's
// signIn() calls it, as the callback page would.
// What the signed-in player has earned: the fake's presence makes it up, the same career its card shows.
export type FakeCareer = {
    ownDashboard(): TitleDashboard
    ownFronts(): Fronts
    ownLevel(): PlayerLevel
    wear(id: string): PlayerTitle | undefined
}

// The server's ladder.
const FAKE_UNLOCKS = {spreadClicks: 2, shields: 3, encloseClicks: 5, bomb: 7, virus: 10, mine: 13, rally: 15}

// The server's scale: what each source of XP but the missions pays.
export const FAKE_XP_RATES: XpRates = {
    tilesTaken: {xp: 1, per: 1},
    daysPlayed: {xp: 100, per: 1},
    bombsDropped: {xp: 50, per: 1},
    boxesCaught: {xp: 30, per: 1},
    rightAnswers: {xp: 30, per: 1},
    tilesFortified: {xp: 1, per: 5},
}

export class FakeAccountBackend implements AccountBackend, PlayerBackend, ProgressionBackend, LevelReader {
    constructor(private readonly career?: FakeCareer) {
    }

    private offered: Provider[] = []
    private linked: Provider[] = []
    private name = ""
    private color = NameColor.UNSPECIFIED
    // The account deleteAccount() asked to delete: the next sign-in brings it back, as the server would within 30 days.
    private leaving?: {linked: Provider[], name: string, color: NameColor}

    public offerSignIn(): void {
        this.offered = ["google", "discord", "email"]
    }

    public async signInOptions(): Promise<Provider[]> {
        return [...this.offered]
    }

    public async me(): Promise<Me> {
        return {linked: [...this.linked]}
    }

    // Straight back to the callback page, as the provider would after the player agreed.
    public async startSignIn(provider: OAuthProvider): Promise<string> {
        if (!this.offered.includes(provider)) throw new AuthError("off")
        return `${CALLBACK_PATH}?code=${encodeURIComponent("Fake Nova")}&state=fake`
    }

    public async completeSignIn(name: string): Promise<SignInOutcome> {
        if (this.restore()) return "restored"
        const outcome = this.linked.length === 0 ? "linked" : "signedIn"
        this.linked = ["google"]
        this.name = name
        return outcome
    }

    public async startEmailSignIn(): Promise<void> {
        throw new AuthError("off")
    }

    public async completeEmailSignIn(): Promise<SignInOutcome> {
        throw new AuthError("off")
    }

    public async signOut(): Promise<void> {
        this.forget()
    }

    public async signOutEverywhere(): Promise<void> {
        this.forget()
    }

    public async deleteAccount(): Promise<void> {
        this.leaving = {linked: this.linked, name: this.name, color: this.color}
        this.forget()
    }

    public async profile(): Promise<ColoredProfile> {
        return {accountId: FAKE_YOU, name: this.name, color: this.color}
    }

    // What GetNameplates answers for the page's own account once it has a username, as the server would.
    public nameplateOf(accountId: string): Nameplate | undefined {
        if (accountId !== FAKE_YOU || this.name === "") return undefined
        return {name: this.name, color: this.color, admin: false, guest: false}
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
        return this.career?.ownDashboard() ?? {wearable: [], tracks: []}
    }

    public async wearTitle(id: string): Promise<PlayerTitle | undefined> {
        this.requireSignedIn()
        return this.career?.wear(id)
    }

    public async fronts(): Promise<Fronts> {
        return this.career?.ownFronts() ?? {playsFor: [], playsAgainst: []}
    }

    // A guest has no level, but the ladder all the same.
    public async readLevel(): Promise<LevelRead> {
        return {
            level: this.linked.length > 0 ? this.career?.ownLevel() : undefined,
            unlocks: FAKE_UNLOCKS,
            missionRates: FAKE_MISSION_RATES,
            xpRates: FAKE_XP_RATES,
        }
    }

    private requireSignedIn() {
        if (this.linked.length === 0) throw new PlayerError("guest")
    }

    private restore(): boolean {
        const leaving = this.leaving
        if (!leaving) return false
        this.linked = leaving.linked
        this.name = leaving.name
        this.color = leaving.color
        this.leaving = undefined
        return true
    }

    private forget() {
        this.linked = []
        this.name = ""
        this.color = NameColor.UNSPECIFIED
    }
}
