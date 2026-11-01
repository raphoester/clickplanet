import {ReactNode, useEffect, useLayoutEffect, useRef, useState} from "react"
import {Season} from "../../backends/season.ts"
import {Countries} from "../../domain/countries.ts"
import {firstsOf} from "../../domain/seasonStats.ts"
import {spanLine} from "../../domain/seasonClock.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {TrophyIcon} from "../components/icons.tsx"
import SeasonStats, {SeasonStatsProps} from "./SeasonStats.tsx"
import {ARCHIVE_PATH} from "../archive/archiveLink.ts"
import "./HallOfFame.css"

export type HallOfFameSeasons = {
    // The season that runs, or else the next one.
    teased?: Season
    ended: Season[]
    standings: SeasonStatsProps["standings"]
    turn: SeasonStatsProps["turn"]
}

export function HallOfFameButton(props: HallOfFameSeasons) {
    const [open, setOpen] = useState(false)

    return <>
        <button type="button" className="button button-mini button-gold hall-of-fame-button" onClick={() => setOpen(true)}>
            <TrophyIcon size={16}/>
            <span>Hall of Fame</span>
        </button>
        {open && <HallOfFame {...props} onClose={() => setOpen(false)}/>}
    </>
}

export default function HallOfFame({teased, ended, standings, turn, onClose}: HallOfFameSeasons & {onClose: () => void}) {
    const [viewing, setViewing] = useState<Season>()

    return <Modal title={viewing ? `Season ${viewing.number}` : "Hall of Fame"} className="hall-of-fame" onClose={onClose}>
        {viewing
            ? <SeasonStats season={viewing} standings={standings} turn={turn} onBack={() => setViewing(undefined)}/>
            : <Plaques teased={teased} ended={ended} standings={standings} onView={setViewing}/>}
    </Modal>
}

// Oldest first, opened on the newest.
function Plaques({teased, ended, standings, onView}: Pick<HallOfFameSeasons, "teased" | "ended" | "standings"> & {
    onView: (season: Season) => void,
}) {
    const seasons = useRef<HTMLOListElement>(null)

    useLayoutEffect(() => {
        const list = seasons.current
        if (list) list.scrollLeft = list.scrollWidth
    }, [])

    return <ol ref={seasons} className="hall-of-fame-seasons">
        {ended.map((season) => <Plaque key={season.number} season={season}>
            <Winner standings={standings} season={season}/>
            <span className="hall-of-fame-view">
                <button type="button" className="button button-mini button-secondary" onClick={() => onView(season)}>View</button>
                <a className="button button-mini button-secondary" href={ARCHIVE_PATH}>Map</a>
            </span>
        </Plaque>)}
        {teased && <Plaque season={teased} teased>
            <span className="hall-of-fame-winner">
                <span className="hall-of-fame-unknown" aria-hidden="true">?</span>
                <span className="sr-only">Not won yet</span>
            </span>
            <span className="hall-of-fame-view" aria-hidden="true"/>
        </Plaque>}
    </ol>
}

function Plaque({season, teased, children}: {season: Season, teased?: boolean, children: ReactNode}) {
    return <li className={teased ? "panel-box hall-of-fame-plaque hall-of-fame-plaque--teased" : "panel-box hall-of-fame-plaque"}>
        <div className="hall-of-fame-plaque-head">
            <span className="hall-of-fame-season">Season {season.number}</span>
            <span className="hall-of-fame-dates">{spanLine(season)}</span>
        </div>
        {children}
    </li>
}

// Every country first in the season's table: level on everything, they won together.
function Winner({standings, season}: {standings: HallOfFameSeasons["standings"], season: Season}) {
    const firsts = useFirsts(standings, season)
    if (firsts === "reading") return <span className="hall-of-fame-winner"/>
    if (firsts.length === 0) return <span className="hall-of-fame-winner hall-of-fame-nobody">No winner</span>

    return <span className="hall-of-fame-winner"
                 role="img"
                 aria-label={firsts.map((code) => Countries.get(code)?.name ?? code).join(" and ")}>
        {firsts.map((code) => <CountryFlag key={code} code={code}/>)}
    </span>
}

function useFirsts(standings: HallOfFameSeasons["standings"], season: Season): string[] | "reading" {
    const [read, setRead] = useState<{bookId: string, firsts: string[]}>()

    useEffect(() => {
        let stale = false
        standings.scores(season.bookId).then(
            (table) => {
                if (!stale) setRead({bookId: season.bookId, firsts: firstsOf(table)})
            },
            (e) => console.error("Could not read the season's table", e),
        )
        return () => {
            stale = true
        }
    }, [standings, season.bookId])

    return read?.bookId === season.bookId ? read.firsts : "reading"
}
