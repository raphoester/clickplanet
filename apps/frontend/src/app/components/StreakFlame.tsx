import {Reaction} from "../../backends/chat.ts"
import {streakShown} from "../../domain/streak.ts"
import {REACTION_IMAGES} from "../chat/reactionsAsset.ts"
import "./StreakFlame.css"

const FIRE = REACTION_IMAGES.get(Reaction.FIRE)?.url

export default function StreakFlame({days, size = 13}: {days: number, size?: number}) {
    if (!streakShown(days)) return null

    const label = `${days}-day streak`
    return <span className="streak-flame" role="img" aria-label={label} title={label}>
        {FIRE && <img src={FIRE} alt="" width={size} height={size} draggable={false}/>}
        <span className="streak-flame-days">{days}</span>
    </span>
}
