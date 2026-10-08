import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {ClosedRound, RoundStanding} from "../../backends/standings.ts"
import {Countries} from "../../domain/countries.ts"
import {
    Move,
    moveOf,
    podiumOf,
    RevealRow,
    revealRowsOf,
    ROUND_REVEAL,
    roundOverLine,
} from "../../domain/roundReveal.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {useModalDialog} from "../components/useDialog.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import "../titles/TitleUnlocked.css"
import "./RoundReveal.css"

const COUNT_STEPS = 24

export type RoundRevealProps = {
    closed: ClosedRound
    countryCode: string
    play: PlaySound
    onClose: () => void
}

type Stage = "podium" | "table"

export default function RoundReveal({closed, countryCode, play, onClose}: RoundRevealProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const [podium] = useState(() => podiumOf(closed))
    const [rows] = useState(() => revealRowsOf(closed, countryCode))
    const [stage, setStage] = useState<Stage>(podium.length > 0 ? "podium" : "table")
    const [moved, setMoved] = useState(false)
    const [ready, setReady] = useState(false)
    const last: Stage = rows.length > 0 ? "table" : "podium"

    useEffect(() => {
        play("title")
    }, [play])

    useEffect(() => {
        if (stage !== "podium") return
        const timer = setTimeout(() => last === "table" ? setStage("table") : setReady(true), ROUND_REVEAL.podium * 1000)
        return () => clearTimeout(timer)
    }, [stage, last])

    useEffect(() => {
        if (stage !== "table") return
        const slide = setTimeout(() => setMoved(true), ROUND_REVEAL.slide * 1000)
        const settle = setTimeout(() => setReady(true), (ROUND_REVEAL.slide + ROUND_REVEAL.settle) * 1000)
        return () => {
            clearTimeout(slide)
            clearTimeout(settle)
        }
    }, [stage])

    useModalDialog(dialog, () => {
        if (ready) onClose()
    })

    return createPortal(
        <div ref={dialog}
             className="title-reveal round-reveal"
             role="dialog"
             aria-modal="true"
             aria-labelledby={headId}
             tabIndex={-1}
             style={{"--steps": podium.length} as CSSProperties}>
            <div className="round-reveal-stage">
                <h2 id={headId} className="title-reveal-head round-reveal-head">{roundOverLine(closed)}</h2>

                {stage === "podium" && <Podium podium={podium}/>}
                {stage === "table" && <Table season={closed.season} rows={rows} moved={moved}/>}

                <div className="title-reveal-actions round-reveal-actions">
                    {!ready && stage === "podium" && last === "table" &&
                        <button type="button" className="button" onClick={() => setStage("table")}>Next</button>}
                    {ready && <button type="button" className="button button-gold" onClick={onClose}>Close</button>}
                </div>
            </div>
            {stage === "podium" && <div className="title-reveal-flash round-reveal-flash" aria-hidden="true"/>}
        </div>,
        document.body,
    )
}

function Podium({podium}: {podium: RoundStanding[]}) {
    const [first, second, third] = podium
    const shown = [second, first, third].filter((standing): standing is RoundStanding => standing !== undefined)

    return <ol className="round-reveal-podium">
        {shown.map((standing) => <li key={standing.countryCode}
                                     className={`round-reveal-step round-reveal-step-${Math.min(standing.rank, 3)}`}
                                     style={{"--rise": podium.length - 1 - podium.indexOf(standing)} as CSSProperties}>
            {standing === first && <div className="title-reveal-rays round-reveal-rays" aria-hidden="true"/>}
            <span className="round-reveal-flag"><CountryFlag code={standing.countryCode}/></span>
            <span className="round-reveal-country">{nameOf(standing.countryCode)}</span>
            <span className="round-reveal-gain">+{standing.points}</span>
            <span className="round-reveal-block" aria-label={`Rank ${standing.rank}`}>
                <span aria-hidden="true">{standing.rank}</span>
            </span>
        </li>)}
    </ol>
}

function Table({season, rows, moved}: {season: number, rows: RevealRow[], moved: boolean}) {
    return <>
        <p className="round-reveal-season">Season {season}</p>
        <ol className="round-reveal-table" style={{"--rows": rows.length} as CSSProperties}>
            {rows.map((row) => {
                const move = moveOf(row)
                return <li key={row.countryCode}
                           className={rowClass(row, move, moved)}
                           style={{"--slot": moved ? row.to : row.from, "--enter": row.from} as CSSProperties}>
                    <span className="round-reveal-rank">{moved ? row.after.rank : row.before?.rank ?? "–"}</span>
                    <CountryFlag code={row.countryCode}/>
                    <span className="round-reveal-name">{nameOf(row.countryCode)}</span>
                    {moved && row.gained > 0 && <span className="round-reveal-gained">+{row.gained}</span>}
                    <span className="round-reveal-points">
                        <CountUp from={row.before?.points ?? 0} to={row.after.points} run={moved}/>
                    </span>
                    <span className="round-reveal-move">{moved && <MoveMark move={move}/>}</span>
                </li>
            })}
        </ol>
    </>
}

function MoveMark({move}: {move: Move}) {
    switch (move.kind) {
        case "up":
            return <span className="round-reveal-up" aria-label={`Up ${move.by}`}>▲{move.by}</span>
        case "down":
            return <span className="round-reveal-down" aria-label={`Down ${move.by}`}>▼{move.by}</span>
        case "new":
            return <span className="round-reveal-new">New</span>
        case "same":
            return <span className="round-reveal-same" aria-label="Same rank">–</span>
    }
}

function CountUp({from, to, run}: {from: number, to: number, run: boolean}) {
    const [value, setValue] = useState(from)

    useEffect(() => {
        if (!run) return
        let step = 0
        const timer = setInterval(() => {
            step += 1
            setValue(Math.round(from + (to - from) * eased(step / COUNT_STEPS)))
            if (step >= COUNT_STEPS) clearInterval(timer)
        }, ROUND_REVEAL.settle * 1000 / 2 / COUNT_STEPS)
        return () => clearInterval(timer)
    }, [from, to, run])

    return <>{value}</>
}

function eased(k: number): number {
    return 1 - (1 - Math.min(k, 1)) ** 3
}

function rowClass(row: RevealRow, move: Move, moved: boolean): string {
    return [
        "round-reveal-row",
        row.mine && "round-reveal-row-mine",
        moved && move.kind === "up" && "round-reveal-row-up",
        moved && move.kind === "new" && "round-reveal-row-new",
    ].filter(Boolean).join(" ")
}

function nameOf(code: string): string {
    return Countries.get(code)?.name ?? code
}
