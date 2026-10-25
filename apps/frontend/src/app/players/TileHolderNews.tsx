import {AnimationEvent} from "react"
import {PlayerLine} from "../../backends/player.ts"
import {agoOf} from "../../domain/tilePeek.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import TitleEmblem from "../titles/TitleEmblem.tsx"
import {truncate} from "../truncate.ts"
import "../components/BombNews.css"
import "../titles/Titles.css"
import "./PlayerCard.css"
import "./TileHolderNews.css"

const NAME_MAX_LENGTH = 16

export type TileHolderNewsProps = {
    line: PlayerLine
    takenAt: number
    now?: number
    lowered?: boolean
    onOpenPlayer?: (player: PlayerLine) => void
    onDone: () => void
}

export default function TileHolderNews({line, takenAt, now = Date.now(), lowered, onOpenPlayer, onDone}: TileHolderNewsProps) {
    const name = truncate(line.name, NAME_MAX_LENGTH)
    const ended = (event: AnimationEvent) => {
        if (event.animationName === "tile-holder-news") onDone()
    }

    return <div className={`bomb-news${lowered ? " bomb-news--lowered" : ""}`} role="status" aria-live="polite">
        <div className="bomb-news-line tile-holder-news-line panel" style={authorStyle(line)} onAnimationEnd={ended}>
            <CountryFlag code={line.countryCode}/>
            {onOpenPlayer
                ? <button type="button"
                          className="tile-holder-news-name player-name-button"
                          title={line.name}
                          onClick={() => onOpenPlayer(line)}>{name}</button>
                : <span className="tile-holder-news-name" title={line.name}>{name}</span>}
            {line.admin && <AdminCrown/>}
            {line.wornTitle && <>
                <span className="title-badge"><TitleEmblem title={line.wornTitle} size={18}/></span>
                <span className="tile-holder-news-title">{line.wornTitle.name}</span>
            </>}
            <span className="tile-holder-news-ago">· {agoOf(takenAt, now)}</span>
        </div>
    </div>
}
