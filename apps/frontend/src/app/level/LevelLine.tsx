import LevelChip from "./LevelChip.tsx"
import XpBar from "./XpBar.tsx"
import XpHelp from "./XpHelp.tsx"
import {useLevel} from "./levelStore.ts"
import "./Level.css"

// The page's own level and its bar, under the clicks and the bonuses. Nothing while it has none.
export default function LevelLine() {
    const own = useLevel((store) => store.own)
    if (!own) return null
    return <div className="level-line">
        <LevelChip level={own.level}/>
        <XpBar level={own}/>
        <XpHelp level={own}/>
    </div>
}
