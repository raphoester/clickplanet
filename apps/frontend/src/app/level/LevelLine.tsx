import {barOf} from "../../domain/level.ts"
import {t, translator} from "../../i18n/page.ts"
import LevelChip from "./LevelChip.tsx"
import XpBar from "./XpBar.tsx"
import XpGainPop from "./XpGainPop.tsx"
import XpHelp from "./XpHelp.tsx"
import {useLevel} from "./levelStore.ts"
import "./Level.css"

// The page's own level, the XP on its bar and the bar, under the clicks and the bonuses. Nothing while it has none. A
// gain the stream tells shows in the XP's place for a moment: how much, and why.
export default function LevelLine() {
    const own = useLevel((store) => store.own)
    const pop = useLevel((store) => store.pops.shown)
    if (!own) return null

    const bar = barOf(own)
    const words = translator()
    return <div className={pop ? "level-line level-line--gain" : "level-line"}>
        <LevelChip level={own.level}/>
        {pop
            ? <XpGainPop key={pop.key} pop={pop}/>
            : <span className="level-line-xp" aria-hidden="true">
                {t("level.xp", {gained: words.number(bar.gained), span: words.number(bar.span)})}
            </span>}
        <XpBar level={own}/>
        <XpHelp level={own}/>
    </div>
}
