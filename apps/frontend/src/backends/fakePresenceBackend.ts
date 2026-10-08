import {OWN_GUEST_NAME} from "./fakeChatBackend.ts"
import {compareRosterEntries} from "../domain/roster.ts"
import {
    CountryTiles,
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

type FakeRank = {id: string, name: string, threshold: number}

const TRACKS: {id: string, name: string, measure: "tiles" | "streak", ranks: FakeRank[]}[] = [
    {id: "conquest", name: "Conquest", measure: "tiles", ranks: [
        {id: "settler", name: "Settler", threshold: 100},
        {id: "raider", name: "Raider", threshold: 1_000},
        {id: "warlord", name: "Warlord", threshold: 10_000},
        {id: "conqueror", name: "Conqueror", threshold: 100_000},
        {id: "warmaster", name: "Warmaster", threshold: 1_000_000},
    ]},
    {id: "devotion", name: "Devotion", measure: "streak", ranks: [
        {id: "loyal", name: "Loyal", threshold: 7},
        {id: "devoted", name: "Devoted", threshold: 30},
        {id: "unbroken", name: "Unbroken", threshold: 100},
    ]},
]

const OG_TITLE: PlayerTitle = {id: "og", name: "OG"}

function rankedTitle(trackIndex: number, rankIndex: number): PlayerTitle {
    const track = TRACKS[trackIndex]
    const rank = track.ranks[rankIndex]
    return {
        id: rank.id,
        name: rank.name,
        rank: {trackId: track.id, trackName: track.name, number: rankIndex + 1, count: track.ranks.length},
    }
}

function shownTitles(tilesTaken: number, streakBest: number, createdAt: number): PlayerTitle[] {
    const shown = createdAt < Date.UTC(2026, 10, 1) ? [OG_TITLE] : []
    TRACKS.forEach((track, trackIndex) => {
        const reached = track.measure === "tiles" ? tilesTaken : streakBest
        const highest = track.ranks.filter((rank) => reached >= rank.threshold).length - 1
        if (highest >= 0) shown.push(rankedTitle(trackIndex, highest))
    })
    return shown
}

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
    private readonly titleListeners = new Set<(title: PlayerTitle) => void>()

    constructor(private readonly now: () => number = () => Date.now()) {
    }

    public heldSession(): string | undefined {
        return "fake-session"
    }

    public heldIdentity(): string | undefined {
        return "fake-session"
    }

    public async announce(presence: Presence): Promise<boolean> {
        this.own = {presence, at: this.now()}
        return true
    }

    public leave(): void {
        this.own = undefined
    }

    public earnTitle(id = "conqueror"): void {
        const trackIndex = TRACKS.findIndex((track) => track.ranks.some((rank) => rank.id === id))
        const title = trackIndex === -1
            ? OG_TITLE
            : rankedTitle(trackIndex, TRACKS[trackIndex].ranks.findIndex((rank) => rank.id === id))
        this.titleListeners.forEach((listener) => listener(title))
    }

    public listenForRoster(
        onEvent: (event: RosterEvent) => void,
        _onUnavailable: () => void,
        onTitleEarned: (title: PlayerTitle) => void,
    ): () => void {
        this.titleListeners.add(onTitleEarned)
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
        return () => {
            clearInterval(timer)
            this.titleListeners.delete(onTitleEarned)
        }
    }

    private roster(): RosterEntry[] {
        const now = this.now()
        const entries = PLAYERS
            .filter((_, index) => onShift(index, now))
            .map((entry) => entry.guest ? entry : {...entry, wornTitle: this.career(entry).wornTitle})

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

        return {
            ...this.career(player),
            name: player.name,
            streakCurrent: player.streak,
            admin: player.admin,
            color: player.color,
        }
    }

    private career(player: RosterEntry) {
        const seed = [...player.name].reduce((sum, c) => sum * 31 + c.charCodeAt(0), 7) >>> 0
        const tilesTaken = seed % 25_000
        const streakBest = player.streak + seed % 40
        const createdAt = this.now() - (1 + seed % 200) * 86_400_000
        const titles = shownTitles(tilesTaken, streakBest, createdAt)
        return {
            tilesTaken, streakBest, createdAt, titles, wornTitle: titles.find((title) => title.rank) ?? titles[0],
            playsFor: fronts([player.countryCode, ...rivals(player.countryCode, seed, seed % 4)], tilesTaken, 6),
            playsAgainst: fronts(rivals(player.countryCode, seed >>> 3, 2 + seed % 9), Math.floor(tilesTaken * 0.7), 2),
        }
    }
}

const RIVALS = ["de", "es", "it", "gb", "be", "pt", "nl", "pl", "br", "us", "jp", "in", "ru", "ca", "mx", "ch"]

function rivals(own: string, seed: number, count: number): string[] {
    const others = RIVALS.filter((code) => code !== own)
    return Array.from({length: count}, (_, i) => others[(seed + i * 7) % others.length])
}

function fronts(countries: string[], tiles: number, steepness: number): CountryTiles[] {
    const weights = countries.map((_, i) => 1 / steepness ** i)
    const total = weights.reduce((sum, weight) => sum + weight, 0)
    return countries
        .map((countryCode, i) => ({countryCode, tiles: Math.max(1, Math.round(tiles * weights[i] / total))}))
        .filter((country) => tiles > 0 && country.tiles > 0)
}

function onShift(index: number, now: number): boolean {
    return Math.floor(now / SHIFT_MS + index * 0.7) % 3 !== 0
}
