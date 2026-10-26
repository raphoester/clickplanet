import {ChargeKind} from "../domain/bonus.ts"
import {AllegianceSource, TileClicker} from "./backend.ts"
import {DayMissions, Mission, MissionReward, MissionsBackend, TileOrder} from "./missions.ts"

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
                if (countryId === this.flag) this.taken++
                if (countryId === this.flag && this.primed) this.finished = true
                this.payDone()
            },
        }
    }

    // As on the server, a take finishes the missions and pays them: finishing them alone would pay with no take to show.
    public prime(): void {
        this.primed = true
    }

    public async missions(): Promise<DayMissions> {
        const flag = this.flag ?? this.pledged
        return {
            countryCode: flag ?? "",
            enlisted: this.flag !== undefined,
            endsAt: this.endOfDay(),
            missions: flag ? this.brief(flag) : [],
        }
    }

    private payDone(): void {
        const flag = this.flag ?? this.pledged
        if (!flag) return
        for (const mission of this.brief(flag)) {
            if (!mission.done || this.paid.has(mission.id)) continue
            this.paid.add(mission.id)
            this.grant(mission.reward.kind)
        }
    }

    private brief(flag: string): Mission[] {
        const rival = flag === "dz" ? "pw" : "dz"
        const taken = (goal: number, share: number) => {
            if (this.finished) return goal
            const counted = Math.min(goal, Math.floor(this.taken * share))
            return this.primed ? Math.max(counted, goal - 1) : counted
        }
        const mission = (id: string, order: Omit<TileOrder, "taken">, reward: MissionReward, share: number): Mission => {
            const tiles = taken(order.goal, share)
            return {id, order: {...order, taken: tiles}, reward, done: tiles >= order.goal, paid: this.paid.has(id)}
        }

        return [
            mission("bank", {kind: "takeFor", country: flag, goal: 60}, {kind: "shields", amount: 10}, 1),
            mission("rival", {kind: "takeFrom", country: rival, goal: 40}, {kind: "spreadClicks", amount: 4}, 0.5),
            mission("ground", {kind: "takeIn", country: "br", goal: 200}, {kind: "bomb", amount: 1}, 0.25),
        ]
    }

    private endOfDay(): number {
        const now = this.now()
        const cutoff = Math.floor(now / DAY_MS) * DAY_MS + CUTOFF_MS
        return now < cutoff ? cutoff : cutoff + DAY_MS
    }
}
