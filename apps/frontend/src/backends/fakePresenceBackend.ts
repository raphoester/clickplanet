import {OWN_GUEST_NAME} from "./fakeChatBackend.ts"
import {compareRosterEntries} from "../domain/roster.ts"
import {PlayerInfo, PlayerInfoBackend, Presence, PresenceBackend, RosterEntry, RosterEvent} from "./player.ts"

/** How long one of the fake players stays on, or off, before it may flip. */
const SHIFT_MS = 25_000

/** How often the fake looks for a player who came or went. */
const TICK_MS = 1_000

/** The line this browser gets. */
const OWN_KEY = "0"

/** The chat's fake chatters and a few who only play, named as the chat names them. */
const PLAYERS: RosterEntry[] = [
    {key: "1", name: "Ana", countryCode: "fr", guest: false, admin: true},
    {key: "2", name: "kiran_07", countryCode: "in", guest: false, admin: false},
    {key: "3", name: "Mateus", countryCode: "br", guest: false, admin: false},
    {key: "4", name: "zoe_nz", countryCode: "nz", guest: false, admin: false},
    {key: "5", name: "guest_91aa3d", countryCode: "de", guest: true, admin: false},
    {key: "6", name: "guest_aa1290", countryCode: "jp", guest: true, admin: false},
    {key: "7", name: "guest_3b7f02", countryCode: "us", guest: true, admin: false},
    {key: "8", name: "guest_7e21c9", countryCode: "ng", guest: true, admin: false},
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

    public leave(): void {
        this.own = undefined
    }

    /** The whole roster at once, then a change each time a shift turns or this browser comes or goes. */
    public listenForRoster(onEvent: (event: RosterEvent) => void): () => void {
        let shown = this.roster()
        onEvent({kind: "roster", entries: shown})

        const timer = setInterval(() => {
            const next = this.roster()
            for (const entry of shown) {
                if (!next.some((e) => e.key === entry.key)) onEvent({kind: "left", key: entry.key})
            }
            for (const entry of next) {
                const was = shown.find((e) => e.key === entry.key)
                if (!was || JSON.stringify(was) !== JSON.stringify(entry)) onEvent({kind: "entry", entry})
            }
            shown = next
        }, TICK_MS)
        return () => clearInterval(timer)
    }

    private roster(): RosterEntry[] {
        const now = this.now()
        const entries = PLAYERS.filter((_, index) => onShift(index, now))

        // Gone 90s after its last announce, as on the server.
        if (this.own && now - this.own.at < 90_000) {
            const {countryCode} = this.own.presence
            entries.push({key: OWN_KEY, name: OWN_GUEST_NAME, countryCode, guest: true, admin: false})
        }

        return entries.sort(compareRosterEntries)
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
