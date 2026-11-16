import {barOf} from "../../domain/level.ts"
import {t, translator} from "../../i18n/page.ts"
import LevelChip from "./LevelChip.tsx"
import XpBar from "./XpBar.tsx"
import XpHelp from "./XpHelp.tsx"
import {useLevel} from "./levelStore.ts"
import "./Level.css"

// The page's own level, the XP on its bar and the bar, under the clicks and the bonuses. Nothing while it has none.
export default function LevelLine() {
    const own = useLevel((store) => store.own)
    if (!own) return null

    const bar = barOf(own)
    const words = translator()
    return <div className="level-line">
        <LevelChip level={own.level}/>
        <span className="level-line-xp" aria-hidden="true">
            {t("level.xp", {gained: words.number(bar.gained), span: words.number(bar.span)})}
        </span>
        <XpBar level={own}/>
        <XpHelp level={own}/>
    </div>
}
