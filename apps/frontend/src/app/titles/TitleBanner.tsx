import {PlayerTitle} from "../../backends/player.ts"
import TitleEmblem from "./TitleEmblem.tsx"
import {rankLine} from "./titleArt.ts"
import "./Titles.css"

export type TitleBannerProps = {
    title: PlayerTitle
    compact?: boolean
}

export default function TitleBanner({title, compact = false}: TitleBannerProps) {
    const line = rankLine(title)

    return <div className={`panel-box title-banner${compact ? " title-banner-compact" : ""}`}>
        <TitleEmblem title={title} size={compact ? 52 : 72} ribbon/>
        <div className="title-banner-text">
            {line && <div className="title-banner-rank">{line}</div>}
            <div className="title-banner-name">{title.name}</div>
        </div>
    </div>
}
