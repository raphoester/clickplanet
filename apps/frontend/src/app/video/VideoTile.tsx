import {FilmIcon} from "../components/icons.tsx"
import "./VideoTile.css"

export type VideoTileProps = {
    dot: boolean
    onMake: () => void
}

export default function VideoTile({dot, onMake}: VideoTileProps) {
    return <button type="button" className="panel-box menu-tile video-tile" onClick={onMake}>
        <FilmIcon size={22}/>
        <span>Make my video</span>
        {dot && <span className="video-dot" aria-hidden="true"/>}
    </button>
}

export function VideoDot() {
    return <span className="video-dot" aria-hidden="true"/>
}
