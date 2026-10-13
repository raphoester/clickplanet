import {ShareStats} from "../../domain/shareCard.ts";
import {CameraIcon} from "../components/icons.tsx";
import RecruitLink from "./RecruitLink.tsx";
import "./SharePlace.css"

export type SharePlaceProps = {
    stats: ShareStats
    onTakePicture?: () => void
    taking?: boolean
}

export default function SharePlace({stats, onTakePicture, taking}: SharePlaceProps) {
    return <div className="share-place">
        {onTakePicture && <button type="button"
                                  className="panel-box menu-tile"
                                  aria-busy={taking}
                                  disabled={taking}
                                  onClick={onTakePicture}>
            <CameraIcon size={22}/>
            <span>Take a picture</span>
        </button>}
        <RecruitLink stats={stats}/>
    </div>
}
