import {ChargeKind} from "../domain/bonus.ts"
import {AllegianceSource, TileClicker} from "./backend.ts"
import {DayMissions, Mission, MissionError, MissionReward, MissionsBackend} from "./missions.ts"

const DAY_MS = 24 * 3600_000
const CUTOFF_MS = 21 * 3600_000

export class FakeMissionsBackend implements MissionsBackend {
    private taken = 0
    private flag?: string
    private pledged?: string
    private finished = false
    private readonly collected = new Set<string>()

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
            },
        }
    }

    public finish(): void {
        this.finished = true
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

    public async collectMission(id: string): Promise<MissionReward> {
        const mission = this.brief(this.flag ?? "").find((m) => m.id === id)
        if (!mission || mission.progress < mission.goal || mission.collected) throw new MissionError("failed")

        this.collected.add(id)
        this.grant(mission.reward.kind)
        return mission.reward
    }

    private brief(flag: string): Mission[] {
        const rival = flag === "dz" ? "pw" : "dz"
        const progress = (goal: number, share: number) => this.finished ? goal : Math.min(goal, Math.floor(this.taken * share))
        const mission = (id: string, rest: Omit<Mission, "id" | "collected" | "progress">, share: number): Mission =>
            ({id, ...rest, progress: progress(rest.goal, share), collected: this.collected.has(id)})

        return [
            mission("bank", {kind: "takeFor", target: flag, goal: 60, reward: {kind: "shields", amount: 10}}, 1),
            mission("rival", {kind: "takeFrom", target: rival, goal: 40, reward: {kind: "spreadClicks", amount: 4}}, 0.5),
            mission("ground", {kind: "takeIn", target: "br", goal: 200, reward: {kind: "bomb", amount: 1}}, 0.25),
        ]
    }

    private endOfDay(): number {
        const now = this.now()
        const cutoff = Math.floor(now / DAY_MS) * DAY_MS + CUTOFF_MS
        return now < cutoff ? cutoff : cutoff + DAY_MS
    }
}
