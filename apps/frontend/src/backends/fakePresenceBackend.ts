import {guestName} from "./chat.ts"
import {PlayerInfo, PlayerInfoBackend, Presence, PresenceBackend, RosterEntry} from "./player.ts"

/** How long one of the fake players stays on, or off, before it may flip. */
const SHIFT_MS = 25_000

/** The tag this browser gets, as in `FakeChatBackend`. */
const OWN_TAG = "c0ffee"

/** The chat's fake chatters and a few who only play. */
const PLAYERS: RosterEntry[] = [
    {name: "Ana", tag: "4f2ca1", countryCode: "fr", guest: false, admin: true},
    {name: "kiran_07", tag: "0c77e2", countryCode: "in", guest: false, admin: false},
    {name: "Mateus", tag: "5d0b19", countryCode: "br", guest: false, admin: false},
    {name: "zoe_nz", tag: "e3a441", countryCode: "nz", guest: false, admin: false},
    {name: guestName("Bo"), tag: "91aa3d", countryCode: "de", guest: true, admin: false},
    {name: guestName("Yuki"), tag: "aa1290", countryCode: "jp", guest: true, admin: false},
    {name: guestName("3b7f02"), tag: "3b7f02", countryCode: "us", guest: true, admin: false},
    {name: guestName("Olu"), tag: "7e21c9", countryCode: "ng", guest: true, admin: false},
]

/**
 * The roster for `VITE_FAKE_BACKEND`: a handful of players and guests, each
 * coming and going on its own shift so the list and its count move in dev.
 * This browser joins once it announces — with no session here, at once, where
 * the real one waits for a click to mint a token.
 */
export class FakePresenceBackend implements PresenceBackend, PlayerInfoBackend {
    private own?: {presence: Presence, at: number}

    constructor(private readonly now: () => number = () => Date.now()) {
    }

    public heldSession(): string | undefined {
        return "fake-session"
    }

    public async announce(presence: Presence): Promise<boolean> {
        this.own = {presence, at: this.now()}
        return true
    }

    public async roster(): Promise<RosterEntry[]> {
        const now = this.now()
        const entries = PLAYERS.filter((_, index) => onShift(index, now))

        // Gone 90s after its last announce, as on the server.
        if (this.own && now - this.own.at < 90_000) {
            const {countryCode, guestName: typed} = this.own.presence
            entries.push({name: guestName(typed || OWN_TAG), tag: OWN_TAG, countryCode, guest: true, admin: false})
        }

        return entries.sort((a, b) =>
            Number(a.guest) - Number(b.guest) || a.name.toLowerCase().localeCompare(b.name.toLowerCase()))
    }

    /** Stats made up from the name, so one player reads the same every time. */
    public async playerInfo(name: string): Promise<PlayerInfo | undefined> {
        const player = PLAYERS.find((p) => !p.guest && p.name.toLowerCase() === name.toLowerCase())
        if (!player) return undefined

        const seed = [...player.name].reduce((sum, c) => sum * 31 + c.charCodeAt(0), 7) >>> 0
        const streakBest = 1 + seed % 40
        return {
            name: player.name,
            tilesTaken: seed % 25_000,
            streakCurrent: seed % 3 === 0 ? 0 : 1 + seed % streakBest,
            streakBest,
            createdAt: this.now() - (1 + seed % 200) * 86_400_000,
            admin: player.admin,
        }
    }
}

/** Off for one shift in three, each player on its own offset. */
function onShift(index: number, now: number): boolean {
    return Math.floor(now / SHIFT_MS + index * 0.7) % 3 !== 0
}
