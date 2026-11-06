import {CSSProperties, useEffect, useId, useRef, useState} from "react"
import {createPortal} from "react-dom"
import {TeaserCard} from "../../backends/promo.ts"
import {BattleToCome} from "../../backends/turn.ts"
import {battleNameOf} from "../../domain/battle.ts"
import {battleWhenLine, teaserClockLine} from "../../domain/battleTeaser.ts"
import {TITLE_REVEAL} from "../../domain/titleReveal.ts"
import {useModalDialog} from "../components/useDialog.ts"
import {useNow} from "../season/useNow.ts"
import {PlaySound} from "../sound/soundPlayer.ts"
import TeaserCarousel from "./TeaserCarousel.tsx"
import "../titles/TitleUnlocked.css"
import "./Battle.css"

export type BattleTeaserProps = {
    battle: BattleToCome
    cards: readonly TeaserCard[]
    // The player opened it: no sound, no slam, and it closes at once.
    asked?: boolean
    play: PlaySound
    onClose: () => void
}

// A battle to come, full screen: when, what it is worth, the operator's words, and its rule set's cards. Asked for, it
// is also the battle that plays, counted down to its end.
export default function BattleTeaser({battle, cards, asked = false, play, onClose}: BattleTeaserProps) {
    const dialog = useRef<HTMLDivElement>(null)
    const headId = useId()
    const [ready, setReady] = useState(asked)
    const now = useNow()

    useEffect(() => {
        if (asked) return
        play("title")
        const timer = setTimeout(() => setReady(true), TITLE_REVEAL.ready * 1000)
        return () => clearTimeout(timer)
    }, [asked, play])

    useModalDialog(dialog, () => {
        if (ready) onClose()
    })

    const timing = {"--impact": `${TITLE_REVEAL.impact * 1000}ms`} as CSSProperties
    const worth = battle.times > 1 ? ` · Points ×${battle.times}` : ""

    return createPortal(
        <div ref={dialog}
             className={asked ? "title-reveal battle-reveal battle-teaser battle-teaser--asked" : "title-reveal battle-reveal battle-teaser"}
             data-mood="battle"
             role="dialog"
             aria-modal="true"
             aria-labelledby={headId}
             tabIndex={-1}
             style={timing}>
            <div className="title-reveal-stage">
                <h2 id={headId} className="title-reveal-head">{battleNameOf(battle)}</h2>
                <p className="battle-brief-times battle-teaser-left" role="timer">
                    <span>{teaserClockLine(battle, now)}</span>
                </p>
                <p className="title-reveal-rank battle-teaser-when">{battleWhenLine(battle, now)}{worth}</p>
                {battle.description !== "" && <p className="battle-brief-text battle-teaser-words">{battle.description}</p>}
                <TeaserCarousel cards={cards}/>
                <div className="title-reveal-actions">
                    {ready && <button type="button" className="button" onClick={onClose}>To the globe</button>}
                </div>
            </div>
            {!asked && <div className="title-reveal-flash" aria-hidden="true"/>}
        </div>,
        document.body,
    )
}
