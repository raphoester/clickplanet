import {CSSProperties, useEffect, useLayoutEffect, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {PlayerLine} from "../../backends/player.ts"
import {cardPlacement} from "../../domain/cardPlacement.ts"
import {Countries} from "../../domain/countries.ts"
import {agoOf, Point} from "../../domain/tilePeek.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {CloseIcon, UserIcon} from "../components/icons.tsx"
import StreakFlame from "../components/StreakFlame.tsx"
import {useEscape} from "../components/useDialog.ts"
import TitleBadge from "../titles/TitleBadge.tsx"
import {truncate} from "../truncate.ts"
import {HolderShown} from "./useTileHolder.ts"
import "./HolderCard.css"

const NAME_MAX_LENGTH = 18
const FLAGS_SHOWN = 3

const CLICK_AFTER_TAP_MS = 600

export type HolderCardProps = {
    holder: HolderShown
    now?: number
    onHold: (held: boolean) => void
    onOpenPlayer?: (player: PlayerLine) => void
    onClose: () => void
}

export default function HolderCard({holder, now = Date.now(), onHold, onOpenPlayer, onClose}: HolderCardProps) {
    const {line, playsFor, mine, takenAt, point, touch} = holder
    const country = Countries.get(line.countryCode)?.name ?? line.countryCode

    const card = useRef<HTMLElement>(null)
    const [place, setPlace] = useState<Point>()
    useLayoutEffect(() => {
        const box = card.current?.getBoundingClientRect()
        if (!box) return
        setPlace(cardPlacement(point, box, {width: window.innerWidth, height: window.innerHeight}, touch))
    }, [point, touch])

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
            <button type="button" className="holder-card-close" aria-label="Close" onClick={onClose}>
                <CloseIcon size={14}/>
            </button>
            <div className="holder-card-main">
                {line.wornTitle && <TitleBadge title={line.wornTitle} size={48}/>}
                <div className="holder-card-who">
                    <span className="holder-card-name">
                        <span className="holder-card-name-text" title={line.name}>{truncate(line.name, NAME_MAX_LENGTH)}</span>
                        {line.admin && <AdminCrown/>}
                        <StreakFlame days={line.streak}/>
                    </span>
                    {playsFor.length > 0 && <span className="holder-card-flags" role="group" aria-label="Fought for">
                        {playsFor.slice(0, FLAGS_SHOWN).map(({countryCode}) => {
                            const name = Countries.get(countryCode)?.name ?? countryCode
                            return <span key={countryCode} role="img" aria-label={name} title={name}><CountryFlag code={countryCode}/></span>
                        })}
                    </span>}
                </div>
                {onOpenPlayer && !line.guest && line.accountId && <button type="button"
                                                         className="icon-button holder-card-profile"
                                                         aria-label="View profile"
                                                         title="View profile"
                                                         onClick={() => onOpenPlayer(line)}>
                    <UserIcon size={20}/>
                </button>}
            </div>
            <p className="holder-card-take">
                <span role="img" aria-label={country} title={country}><CountryFlag code={line.countryCode}/></span>
                {mine ? "You took it" : "Took it"} {agoOf(takenAt, now)}
            </p>
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
