import {ReactNode, useEffect, useLayoutEffect, useRef, useState} from "react"
import {Season} from "../../backends/season.ts"
import {firstsOf} from "../../domain/seasonStats.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import {CrownIcon, TrophyIcon} from "../components/icons.tsx"
import SeasonStats, {SeasonStatsProps} from "./SeasonStats.tsx"
import {ARCHIVE_PATH} from "../archive/archiveLink.ts"
import {seasonSpan} from "./dates.ts"
import "./HallOfFame.css"

export type HallOfFameSeasons = {
    // The season that runs, or else the next one.
    teased?: Season
    ended: Season[]
    standings: SeasonStatsProps["standings"]
    turn: SeasonStatsProps["turn"]
}

// icon: the square button of a panel's header, beside its title, rather than a gold button over the board.
export function HallOfFameButton({icon, ...props}: HallOfFameSeasons & {icon?: boolean}) {
    const [open, setOpen] = useState(false)

    return <>
        {icon
            ? <button type="button"
                      className="icon-button hall-of-fame-icon"
                      aria-label={t("hallOfFame.title")}
                      title={t("hallOfFame.title")}
                      aria-haspopup="dialog"
                      onClick={() => setOpen(true)}>
                <CrownIcon size={20}/>
            </button>
            : <button type="button" className="button button-mini button-gold hall-of-fame-button" onClick={() => setOpen(true)}>
                <TrophyIcon size={16}/>
                <span>{t("hallOfFame.title")}</span>
            </button>}
        {open && <HallOfFame {...props} onClose={() => setOpen(false)}/>}
    </>
}

export default function HallOfFame({teased, ended, standings, turn, onClose}: HallOfFameSeasons & {onClose: () => void}) {
    const [viewing, setViewing] = useState<Season>()

    return <Modal title={viewing ? t("season.name", {season: viewing.number}) : t("hallOfFame.title")} className="hall-of-fame" onClose={onClose}>
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
                <button type="button" className="button button-mini button-secondary" onClick={() => onView(season)}>{t("hallOfFame.view")}</button>
                <a className="button button-mini button-secondary" href={ARCHIVE_PATH}>{t("hallOfFame.map")}</a>
            </span>
        </Plaque>)}
        {teased && <Plaque season={teased} teased>
            <span className="hall-of-fame-winner">
                <span className="hall-of-fame-unknown" aria-hidden="true">?</span>
                <span className="sr-only">{t("hallOfFame.notWonYet")}</span>
            </span>
            <span className="hall-of-fame-view" aria-hidden="true"/>
        </Plaque>}
    </ol>
}

function Plaque({season, teased, children}: {season: Season, teased?: boolean, children: ReactNode}) {
    return <li className={teased ? "panel-box hall-of-fame-plaque hall-of-fame-plaque--teased" : "panel-box hall-of-fame-plaque"}>
        <div className="hall-of-fame-plaque-head">
            <span className="hall-of-fame-season">{t("season.name", {season: season.number})}</span>
            <span className="hall-of-fame-dates">{seasonSpan(season)}</span>
        </div>
        {children}
    </li>
}

// Every country first in the season's table: level on everything, they won together.
function Winner({standings, season}: {standings: HallOfFameSeasons["standings"], season: Season}) {
    const firsts = useFirsts(standings, season)
    if (firsts === "reading") return <span className="hall-of-fame-winner"/>
    if (firsts.length === 0) return <span className="hall-of-fame-winner hall-of-fame-nobody">{t("hallOfFame.noWinner")}</span>

    return <span className="hall-of-fame-winner"
                 role="img"
                 aria-label={translator().list(firsts.map((code) => translator().country(code)))}>
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
