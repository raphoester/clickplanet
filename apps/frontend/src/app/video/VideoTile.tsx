import {FilmIcon} from "../components/icons.tsx"
import {t} from "../../i18n/page.ts"
import "./VideoTile.css"

export type VideoTileProps = {
    dot: boolean
    onMake: () => void
}

export default function VideoTile({dot, onMake}: VideoTileProps) {
    return <button type="button" className="panel-box menu-tile video-tile" onClick={onMake}>
        <FilmIcon size={22}/>
        <span>{t("video.make")}</span>
        {dot && <span className="video-dot" aria-hidden="true"/>}
    </button>
}

export function VideoDot() {
    return <span className="video-dot" aria-hidden="true"/>
}
