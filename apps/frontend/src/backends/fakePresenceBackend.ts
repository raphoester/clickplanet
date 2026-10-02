import {OWN_GUEST_NAME} from "./fakeChatBackend.ts"
import {compareRosterEntries} from "../domain/roster.ts"
import {
    PlayerInfo,
    PlayerInfoBackend,
    PlayerTitle,
    Presence,
    PresenceBackend,
    RosterEntry,
    RosterEvent,
} from "./player.ts"

const SHIFT_MS = 25_000

const TICK_MS = 1_000

const OWN_KEY = "0"

const LADDER: {title: PlayerTitle, tiles: number, streak: number}[] = [
    {title: "settler", tiles: 100, streak: 0},
    {title: "governor", tiles: 1_000, streak: 0},
    {title: "conqueror", tiles: 10_000, streak: 0},
    {title: "emperor", tiles: 100_000, streak: 0},
    {title: "loyal", tiles: 0, streak: 7},
    {title: "devoted", tiles: 0, streak: 30},
    {title: "unbroken", tiles: 0, streak: 100},
]

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

        if (this.own && now - this.own.at < 90_000) {
            const {countryCode} = this.own.presence
            entries.push({key: OWN_KEY, name: OWN_GUEST_NAME, countryCode, guest: true, admin: false})
        }

        return entries.sort(compareRosterEntries)
    }

    public async playerInfo(name: string): Promise<PlayerInfo | undefined> {
        const player = PLAYERS.find((p) => !p.guest && p.name.toLowerCase() === name.toLowerCase())
        if (!player) return undefined

        const seed = [...player.name].reduce((sum, c) => sum * 31 + c.charCodeAt(0), 7) >>> 0
        const streakBest = 1 + seed % 40
        const tilesTaken = seed % 25_000
        return {
            name: player.name,
            tilesTaken,
            streakCurrent: seed % 3 === 0 ? 0 : 1 + seed % streakBest,
            streakBest,
            createdAt: this.now() - (1 + seed % 200) * 86_400_000,
            admin: player.admin,
            titles: LADDER.filter((rung) => tilesTaken >= rung.tiles && streakBest >= rung.streak).map((rung) => rung.title),
        }
    }
}

function onShift(index: number, now: number): boolean {
    return Math.floor(now / SHIFT_MS + index * 0.7) % 3 !== 0
}
