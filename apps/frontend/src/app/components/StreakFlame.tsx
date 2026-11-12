import {useCallback, useEffect, useRef, useState} from "react"
import {Reaction} from "../../backends/chat.ts"
import {streakShown} from "../../domain/streak.ts"
import {REACTION_IMAGES} from "../chat/reactionsAsset.ts"
import Bubble, {BUBBLE_MS} from "./Bubble.tsx"
import {t} from "../../i18n/page.ts"
import "./StreakFlame.css"

const FIRE = REACTION_IMAGES.get(Reaction.FIRE)?.url

type Shown = "pressed" | "hovered"

export default function StreakFlame({days, size = 13}: {days: number, size?: number}) {
    const button = useRef<HTMLButtonElement>(null)
    const [shown, setShown] = useState<Shown | undefined>()
    const hide = useCallback(() => setShown(undefined), [])

    useEffect(() => {
        if (shown !== "pressed") return
        const timer = setTimeout(() => setShown(undefined), BUBBLE_MS)
        return () => clearTimeout(timer)
    }, [shown])

    if (!streakShown(days)) return null

    return <button type="button"
                   ref={button}
                   className="streak-flame-button"
                   onClick={() => setShown((now) => now === "pressed" ? undefined : "pressed")}
                   onPointerEnter={(event) => event.pointerType === "mouse" && setShown((now) => now ?? "hovered")}
                   onPointerLeave={(event) => event.pointerType === "mouse" && setShown((now) => now === "hovered" ? undefined : now)}>
        <span className="streak-flame" role="img" aria-label={t("board.streak", {days})}>
            {FIRE && <img src={FIRE} alt="" width={size} height={size} draggable={false}/>}
            <span className="streak-flame-days">{days}</span>
        </span>
        {shown && <Bubble anchor={button} onLost={hide}>{t("board.streakBubble", {days})}</Bubble>}
    </button>
}
