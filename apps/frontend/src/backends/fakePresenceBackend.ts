import {OWN_GUEST_NAME} from "./fakeChatBackend.ts"
import {compareRosterEntries} from "../domain/roster.ts"
import {
    NameColor,
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

type FakeTitle = PlayerTitle & {
    earnedBy: (tilesTaken: number, streakBest: number, createdAt: number) => boolean
}

const TITLES: FakeTitle[] = [
    {id: "og", name: "OG", earnedBy: (_, __, createdAt) => createdAt < Date.UTC(2026, 10, 1)},
    {id: "settler", name: "Settler", earnedBy: (tiles) => tiles >= 100},
    {id: "governor", name: "Governor", earnedBy: (tiles) => tiles >= 1_000},
    {id: "conqueror", name: "Conqueror", earnedBy: (tiles) => tiles >= 10_000},
    {id: "loyal", name: "Loyal", earnedBy: (_, streak) => streak >= 7},
    {id: "devoted", name: "Devoted", earnedBy: (_, streak) => streak >= 30},
]

const PLAYERS: RosterEntry[] = [
    {key: "1", name: "Ana", countryCode: "fr", guest: false, admin: true, color: NameColor.PINK, streak: 12},
    {key: "2", name: "kiran_07", countryCode: "in", guest: false, admin: false, color: NameColor.UNSPECIFIED, streak: 3},
    {key: "3", name: "Mateus", countryCode: "br", guest: false, admin: false, color: NameColor.TEAL, streak: 1},
    {key: "4", name: "zoe_nz", countryCode: "nz", guest: false, admin: false, color: NameColor.VIOLET, streak: 41},
    {key: "5", name: "guest_91aa3d", countryCode: "de", guest: true, admin: false, color: NameColor.UNSPECIFIED, streak: 0},
    {key: "6", name: "guest_aa1290", countryCode: "jp", guest: true, admin: false, color: NameColor.UNSPECIFIED, streak: 0},
    {key: "7", name: "guest_3b7f02", countryCode: "us", guest: true, admin: false, color: NameColor.UNSPECIFIED, streak: 0},
    {key: "8", name: "guest_7e21c9", countryCode: "ng", guest: true, admin: false, color: NameColor.UNSPECIFIED, streak: 0},
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
            entries.push({
                key: OWN_KEY, name: OWN_GUEST_NAME, countryCode, guest: true, admin: false,
                color: NameColor.UNSPECIFIED, streak: 0,
            })
        }

        return entries.sort(compareRosterEntries)
    }

    public async playerInfo(name: string): Promise<PlayerInfo | undefined> {
        const player = PLAYERS.find((p) => !p.guest && p.name.toLowerCase() === name.toLowerCase())
        if (!player) return undefined

        const seed = [...player.name].reduce((sum, c) => sum * 31 + c.charCodeAt(0), 7) >>> 0
        const tilesTaken = seed % 25_000
        const streakBest = player.streak + seed % 40
        const createdAt = this.now() - (1 + seed % 200) * 86_400_000
        return {
            name: player.name,
            tilesTaken,
            streakCurrent: player.streak,
            streakBest,
            createdAt,
            admin: player.admin,
            color: player.color,
            titles: TITLES.filter((title) => title.earnedBy(tilesTaken, streakBest, createdAt)).map(({id, name}) => ({id, name})),
        }
    }
}

function onShift(index: number, now: number): boolean {
    return Math.floor(now / SHIFT_MS + index * 0.7) % 3 !== 0
}
