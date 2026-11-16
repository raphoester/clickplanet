import {PointerEvent, RefObject, useCallback, useEffect, useId, useRef, useState} from "react"
import {PlayerLevel} from "../../backends/progression.ts"
import {barOf} from "../../domain/level.ts"
import Bubble from "../components/Bubble.tsx"
import {useEscape} from "../components/useDialog.ts"
import {t, translator} from "../../i18n/page.ts"
import "./Level.css"

type Shown = "closed" | "hover" | "pinned"

const WAYS = [
    "level.gain.tiles",
    "level.gain.days",
    "level.gain.missions",
    "level.gain.bombs",
    "level.gain.boxes",
    "level.gain.quizzes",
    "level.gain.fortify",
] as const

// Laid over a level's line, which must be positioned: a mouse over it, or a press, says what the bar is and how to
// gain XP.
export default function XpHelp({level}: {level: PlayerLevel}) {
    const button = useRef<HTMLButtonElement>(null)
    const id = useId()
    const [shown, setShown] = useState<Shown>("closed")
    const close = useCallback(() => setShown("closed"), [])
    const open = shown !== "closed"
    const mouse = (event: PointerEvent) => event.pointerType === "mouse"

    return <>
        <button type="button"
                ref={button}
                className="xp-help-open"
                aria-label={t("level.gain.title")}
                aria-expanded={open}
                aria-controls={open ? id : undefined}
                onPointerEnter={(event) => mouse(event) && setShown((now) => now === "closed" ? "hover" : now)}
                onPointerLeave={(event) => mouse(event) && setShown((now) => now === "hover" ? "closed" : now)}
                onClick={() => setShown((now) => now === "pinned" ? "closed" : "pinned")}/>
        {open && <Help id={id} anchor={button} level={level} onClose={close}/>}
    </>
}

function Help({id, anchor, level, onClose}: {
    id: string,
    anchor: RefObject<HTMLButtonElement>,
    level: PlayerLevel,
    onClose: () => void,
}) {
    useEscape(onClose)

    useEffect(() => {
        const outside = (event: globalThis.PointerEvent) => {
            if (!anchor.current?.contains(event.target as Node | null)) onClose()
        }
        document.addEventListener("pointerdown", outside)
        return () => document.removeEventListener("pointerdown", outside)
    }, [anchor, onClose])

    const bar = barOf(level)
    const words = translator()
    return <Bubble anchor={anchor} id={id} onLost={onClose}>
        <strong>{t("level.xpBar", {level: level.level + 1})}</strong>
        <span className="xp-help-line">{t("level.xp", {gained: words.number(bar.gained), span: words.number(bar.span)})}</span>
        <span className="xp-help-line">{t("level.reward")}</span>
        <strong className="xp-help-title">{t("level.gain.title")}</strong>
        <ul className="xp-help-ways">
            {WAYS.map((way) => <li key={way}>{t(way)}</li>)}
        </ul>
    </Bubble>
}
