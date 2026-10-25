import {CSSProperties, useEffect, useLayoutEffect, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {PlayerInfoBackend, PlayerLine} from "../../backends/player.ts"
import {cardPlacement} from "../../domain/cardPlacement.ts"
import {Countries} from "../../domain/countries.ts"
import {agoOf, Point} from "../../domain/tilePeek.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {CloseIcon} from "../components/icons.tsx"
import StreakFlame from "../components/StreakFlame.tsx"
import {useEscape} from "../components/useDialog.ts"
import {days} from "../days.ts"
import TitleEmblem from "../titles/TitleEmblem.tsx"
import {truncate} from "../truncate.ts"
import {usePlayerInfo} from "./usePlayerInfo.ts"
import {HolderShown} from "./useTileHolder.ts"
import "./HolderCard.css"

const NAME_MAX_LENGTH = 18
const FLAGS_SHOWN = 3

const CLICK_AFTER_TAP_MS = 600

const count = new Intl.NumberFormat()
const month = new Intl.DateTimeFormat(undefined, {month: "short", year: "numeric"})

export type HolderCardProps = {
    holder: HolderShown
    backend: PlayerInfoBackend
    now?: number
    onHold: (held: boolean) => void
    onOpenPlayer?: (player: PlayerLine) => void
    onClose: () => void
}

export default function HolderCard({holder, backend, now = Date.now(), onHold, onOpenPlayer, onClose}: HolderCardProps) {
    const {line, takenAt, point, touch} = holder
    const state = usePlayerInfo(backend, line)
    const info = state.kind === "ready" ? state.info : undefined
    const country = Countries.get(line.countryCode)?.name ?? line.countryCode

    const card = useRef<HTMLElement>(null)
    const [place, setPlace] = useState<Point>()
    useLayoutEffect(() => {
        const box = card.current?.getBoundingClientRect()
        if (!box) return
        setPlace(cardPlacement(point, box, {width: window.innerWidth, height: window.innerHeight}, touch))
    }, [point, touch, state.kind])

    useEscape(onClose)

    useEffect(() => {
        const outside = (event: PointerEvent) => {
            if (card.current?.contains(event.target as Node)) return
            if (event.pointerType !== "mouse") swallowNextClick()
            onClose()
        }
        document.addEventListener("pointerdown", outside, true)
        return () => document.removeEventListener("pointerdown", outside, true)
    }, [onClose])

    const style = {
        ...authorStyle(line),
        left: `${place?.x ?? 0}px`,
        top: `${place?.y ?? 0}px`,
        visibility: place ? undefined : "hidden",
    } as CSSProperties

    return createPortal(
        <section ref={card}
                 className="holder-card panel"
                 style={style}
                 aria-label={line.name}
                 onPointerEnter={() => onHold(true)}
                 onPointerLeave={() => onHold(false)}>
            <div className="holder-card-head">
                {line.wornTitle && <TitleEmblem title={line.wornTitle} size={44}/>}
                <div className="holder-card-who">
                    <span className="holder-card-name">
                        <span className="holder-card-name-text" title={line.name}>{truncate(line.name, NAME_MAX_LENGTH)}</span>
                        {line.admin && <AdminCrown/>}
                        <StreakFlame days={line.streak}/>
                    </span>
                    {line.wornTitle && <span className="holder-card-title">{line.wornTitle.name}</span>}
                </div>
                <button type="button" className="icon-button holder-card-close" aria-label="Close" onClick={onClose}>
                    <CloseIcon/>
                </button>
            </div>

            <p className="holder-card-take">
                <span role="img" aria-label={country} title={country}><CountryFlag code={line.countryCode}/></span>
                Took it {agoOf(takenAt, now)}
            </p>

            {state.kind === "loading" && <p className="holder-card-note" role="status">Loading…</p>}
            {info && <dl className="holder-card-stats">
                <div className="panel-box">
                    <dt>Tiles taken</dt>
                    <dd>{count.format(info.tilesTaken)}</dd>
                </div>
                <div className="panel-box">
                    <dt>Best streak</dt>
                    <dd>{days(info.streakBest)}</dd>
                </div>
                {info.createdAt !== undefined && <div className="panel-box">
                    <dt>Since</dt>
                    <dd>{month.format(info.createdAt)}</dd>
                </div>}
            </dl>}
            {info && info.playsFor.length > 0 && <p className="holder-card-fronts">
                Fought for
                {info.playsFor.slice(0, FLAGS_SHOWN).map(({countryCode}) => {
                    const name = Countries.get(countryCode)?.name ?? countryCode
                    return <span key={countryCode} role="img" aria-label={name} title={name}><CountryFlag code={countryCode}/></span>
                })}
            </p>}

            {onOpenPlayer && !line.guest && <button type="button"
                                                     className="button button-mini button-secondary holder-card-profile"
                                                     onClick={() => onOpenPlayer(line)}>
                View profile
            </button>}
        </section>,
        document.body)
}

// A tap that closes the card is not also a tap on what is under it: on the globe it would take a tile.
function swallowNextClick() {
    const swallow = (event: MouseEvent) => {
        event.stopPropagation()
        event.preventDefault()
    }
    window.addEventListener("click", swallow, {capture: true, once: true})
    setTimeout(() => window.removeEventListener("click", swallow, {capture: true}), CLICK_AFTER_TAP_MS)
}
