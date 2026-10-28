import {ChargeKind} from "../domain/bonus.ts"
import {AllegianceSource, TileClicker} from "./backend.ts"
import {DayMissions, FlagMission, Mission, MissionReward, MissionsBackend, TileOrder} from "./missions.ts"

const DAY_MS = 24 * 3600_000
const CUTOFF_MS = 21 * 3600_000

export class FakeMissionsBackend implements MissionsBackend {
    private taken = 0
    private flag?: string
    private pledged?: string
    private primed = false
    private finished = false
    private readonly paid = new Set<string>()

    constructor(
        private readonly grant: (kind: ChargeKind) => void,
        allegiance: AllegianceSource,
        private readonly now: () => number = Date.now,
    ) {
        allegiance.listenForAllegiance((held) => {
            this.pledged = held?.countryId
        })
    }

    public counting(clicker: TileClicker): TileClicker {
        return {
            clickTile: async (tileId, countryId, switches) => {
                await clicker.clickTile(tileId, countryId, switches)
                this.flag ??= countryId
                this.taken++
                if (this.primed) this.finished = true
                this.payDone()
            },
        }
    }

    // As on the server, a take finishes the missions and pays them: finishing them alone would pay with no take to show.
    public prime(): void {
        this.primed = true
    }

    public async missions(): Promise<DayMissions> {
        this.flag ??= this.pledged
        if (!this.flag) return {endsAt: this.endOfDay(), missions: []}
        return {endsAt: this.endOfDay(), missions: this.brief(this.flag), flagMission: this.flagMission(this.flag)}
    }

    private payDone(): void {
        if (!this.flag) return
        for (const mission of this.brief(this.flag)) {
            if (!mission.done || this.paid.has(mission.id)) continue
            this.paid.add(mission.id)
            this.grant(mission.reward.kind)
        }
    }

    private counted(goal: number, share: number): number {
        if (this.finished) return goal
        const counted = Math.min(goal, Math.floor(this.taken * share))
        return this.primed ? Math.max(counted, goal - 1) : counted
    }

    // The tab's takes stand for the whole flag's shields broken.
    private flagMission(flag: string): FlagMission {
        const broken = this.counted(300, 0.1)
        return {id: "flag", order: {kind: "breakShields", country: rivalOf(flag), goal: 300, broken}, points: 2, done: broken >= 300}
    }

    private brief(flag: string): Mission[] {
        const rival = rivalOf(flag)
        const taken = (goal: number, share: number) => this.counted(goal, share)
        const mission = (id: string, order: Omit<TileOrder, "taken">, reward: MissionReward, share: number): Mission => {
            const tiles = taken(order.goal, share)
            return {id, order: {...order, taken: tiles}, reward, done: tiles >= order.goal, paid: this.paid.has(id)}
        }

        return [
            mission("bank", {kind: "takeFor", country: flag, goal: 180}, {kind: "shields", amount: 20}, 1),
            mission("rival", {kind: "takeFrom", country: rival, goal: 120}, {kind: "spreadClicks", amount: 6}, 0.5),
            mission("ground", {kind: "takeIn", country: "br", goal: 450}, {kind: "bomb", amount: 1}, 0.25),
            mission("more", {kind: "takeIn", country: "ru", goal: 400}, {kind: "encloseClicks", amount: 2}, 0.2),
        ]
    }

    private endOfDay(): number {
        const now = this.now()
        const cutoff = Math.floor(now / DAY_MS) * DAY_MS + CUTOFF_MS
        return now < cutoff ? cutoff : cutoff + DAY_MS
    }
}

function rivalOf(flag: string): string {
    return flag === "dz" ? "pw" : "dz"
}
