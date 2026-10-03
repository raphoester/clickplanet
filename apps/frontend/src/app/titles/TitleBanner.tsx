import {PlayerTitle} from "../../backends/player.ts"
import TitleEmblem from "./TitleEmblem.tsx"
import {METALS, metalOf, rankLine} from "./titleArt.ts"
import "./Titles.css"

export type TitleBannerProps = {
    title: PlayerTitle
    compact?: boolean
}

export default function TitleBanner({title, compact = false}: TitleBannerProps) {
    const metal = metalOf(title)
    const [light, mid, dark] = METALS[metal]
    const line = rankLine(title)

    return <div className={`title-banner title-sheen${compact ? " title-banner-compact" : ""}`} style={{borderColor: `${mid}66`}}>
        <TitleEmblem title={title} size={compact ? 56 : 96} ribbon={!compact}/>
        <div className="title-banner-text">
            {line && <div className="title-banner-rank" style={{color: mid}}>{line}</div>}
            <div className="title-banner-name"
                 style={{color: light, textShadow: `0 2px 0 ${dark}, 0 4px 0 #000000, 0 0 18px ${mid}55`}}>
                {title.name}
            </div>
        </div>
    </div>
}
