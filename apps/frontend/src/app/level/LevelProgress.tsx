import {barOf} from "../../domain/level.ts"
import {t, translator} from "../../i18n/page.ts"
import LevelChip from "./LevelChip.tsx"
import XpBar from "./XpBar.tsx"
import XpHelp from "./XpHelp.tsx"
import {useLevel} from "./levelStore.ts"
import "./Level.css"

// The page's own level in the Progress tab and the profile: the level, what its bar leads to, the bar and the XP on it.
export default function LevelProgress() {
    const own = useLevel((store) => store.own)
    if (!own) return null

    const bar = barOf(own)
    const words = translator()
    return <div className="level-progress">
        <LevelChip level={own.level} className="level-chip--large"/>
        <div className="level-progress-bar">
            <span className="level-progress-line">
                <span className="level-progress-name">{t("level.xpBar", {level: own.level + 1})}</span>
                <span className="level-progress-xp">{t("level.xp", {gained: words.number(bar.gained), span: words.number(bar.span)})}</span>
            </span>
            <XpBar level={own}/>
        </div>
        <XpHelp level={own}/>
    </div>
}
