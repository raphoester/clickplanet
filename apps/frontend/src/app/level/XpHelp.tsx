import {PointerEvent, RefObject, useCallback, useEffect, useId, useRef, useState} from "react"
import {PlayerLevel, XpRates, XpSource} from "../../backends/progression.ts"
import {barOf, nextUnlock} from "../../domain/level.ts"
import Bubble from "../components/Bubble.tsx"
import {useEscape} from "../components/useDialog.ts"
import {t, translator} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import {Key} from "../../i18n/translator.ts"
import BonusName from "./BonusName.tsx"
import {useLevel} from "./levelStore.ts"
import "./Level.css"

type Shown = "closed" | "hover" | "pinned"

// Each way to gain XP and what it pays on one line, in every language.
const HELP_WIDTH_PX = 320

// Every source of the server's scale, with what it pays from GetLevel: a source added there needs its line here.
const WAYS: readonly {source: XpSource, way: Key}[] = [
    {source: "tilesTaken", way: "level.gain.tiles"},
    {source: "daysPlayed", way: "level.gain.firstTake"},
    {source: "missionsDone", way: "level.gain.missions"},
    {source: "bombsDropped", way: "level.gain.bombs"},
    {source: "boxesCaught", way: "level.gain.boxes"},
    {source: "rightAnswers", way: "level.gain.quizzes"},
    {source: "tilesFortified", way: "level.gain.fortify"},
]

// Laid over a level's line, which must be positioned: a mouse over it, or a press, says what the bar is, what the next
// levels give, and how to gain XP.
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

    const unlocks = useLevel((store) => store.unlocks)
    const rates = useLevel((store) => store.xpRates)
    const bar = barOf(level)
    const words = translator()
    const next = nextUnlock(unlocks, level.level)
    const ladder = Object.keys(unlocks).length > 0
    return <Bubble anchor={anchor} id={id} width={HELP_WIDTH_PX} onLost={onClose}>
        <strong>{t("level.xpBar", {level: level.level + 1})}</strong>
        <span className="xp-help-line">{t("level.xp", {gained: words.number(bar.gained), span: words.number(bar.span)})}</span>
        {ladder && <span className="xp-help-line xp-help-unlock">
            {next ? rich("level.nextUnlock", {bonus: <BonusName kind={next.kind}/>, level: next.level}) : t("level.allUnlocked")}
        </span>}
        <span className="xp-help-line">{t("level.levelUpBonus")}</span>
        <strong className="xp-help-title">{t("level.gain.title")}</strong>
        <ul className="xp-help-ways">
            {WAYS.map(({source, way}) => <li key={source}>
                <span>{t(way)}</span>
                <span className="xp-help-rate">{rateOf(source, rates)}</span>
            </li>)}
        </ul>
    </Bubble>
}

// What one of the source's count pays, as the server says: a fortify's 1 per 5 tiles is 0.2 a tile. A mission says its
// own.
function rateOf(source: XpSource, rates: XpRates): string {
    if (source === "missionsDone") return t("level.rate.missions")
    const rate = rates[source]
    if (!rate) return ""
    return t("level.gained", {xp: translator().number(rate.xp / rate.per, {maximumFractionDigits: 2})})
}
