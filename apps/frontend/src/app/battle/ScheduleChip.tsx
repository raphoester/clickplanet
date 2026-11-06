import {finaleOf, nextBattleOf} from "../../domain/battle.ts"
import {reigningChampionOf} from "../../domain/champion.ts"
import Sheet from "../hud/Sheet.tsx"
import SeasonChip, {ChampionChip, SeasonDetails} from "../season/SeasonChip.tsx"
import {useNow} from "../season/useNow.ts"
import BattleChip, {BattleDetails} from "./BattleChip.tsx"
import {runningSeason, useSchedule} from "./scheduleStore.ts"

// The battle that runs, or else the last season's champion, or else the season's clocks.
export default function ScheduleChip({compact}: {compact: boolean}) {
    const battle = useSchedule((schedule) => schedule.battle)
    const season = useSchedule(runningSeason)
    const closed = useSchedule((schedule) => schedule.closed)
    const clocks = useClocks()

    if (battle) return <BattleChip battle={battle} season={season} compact={compact}/>
    const champion = reigningChampionOf(season, closed)
    if (champion) return <ChampionChip champion={champion} compact={compact}/>
    return season ? <SeasonChip season={season} {...clocks} compact={compact}/> : null
}

export function SeasonSheet({onClose}: {onClose: () => void}) {
    const season = useSchedule(runningSeason)
    const clocks = useClocks()

    if (!season) return null
    return <Sheet title={`Season ${season.number}`} onClose={onClose}>
        <SeasonDetails season={season} {...clocks}/>
    </Sheet>
}

export function BattleSheet({onClose}: {onClose: () => void}) {
    const battle = useSchedule((schedule) => schedule.battle)
    const season = useSchedule(runningSeason)

    if (!battle) return null
    return <Sheet title="Battle" onClose={onClose}>
        <BattleDetails battle={battle} season={season}/>
    </Sheet>
}

function useClocks() {
    const now = useNow()
    const turn = useSchedule((schedule) => schedule.turn)
    const battles = useSchedule((schedule) => schedule.battles)
    const dayEndsAt = useSchedule((schedule) => schedule.seasons.today?.endsAt)
    return {nextBattle: nextBattleOf(battles, now), finale: finaleOf(battles, turn, now), dayEndsAt}
}
