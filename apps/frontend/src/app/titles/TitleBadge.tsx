import {PlayerTitle} from "../../backends/player.ts"
import TitleEmblem from "./TitleEmblem.tsx"
import "./Titles.css"

export default function TitleBadge({title, size = 16}: {title?: PlayerTitle, size?: number}) {
    if (!title) return null

    return <span className="title-badge" role="img" aria-label={title.name} title={title.name}>
        <TitleEmblem title={title} size={size}/>
    </span>
}
