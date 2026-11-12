import {ShareAddress, ShareStats} from "../../domain/shareCard.ts";
import {CameraIcon} from "../components/icons.tsx";
import VideoTile, {VideoTileProps} from "../video/VideoTile.tsx";
import RecruitLink from "./RecruitLink.tsx";
import {t} from "../../i18n/page.ts";
import "./SharePlace.css"

export type SharePlaceProps = {
    stats: ShareStats
    address: ShareAddress
    onTakePicture?: () => void
    taking?: boolean
    // Where this browser can make the flag's video.
    video?: VideoTileProps
}

export default function SharePlace({stats, address, onTakePicture, taking, video}: SharePlaceProps) {
    return <div className="share-place">
        {video && <VideoTile {...video}/>}
        {onTakePicture && <button type="button"
                                  className="panel-box menu-tile"
                                  aria-busy={taking}
                                  disabled={taking}
                                  onClick={onTakePicture}>
            <CameraIcon size={22}/>
            <span>{t("share.takePicture")}</span>
        </button>}
        <RecruitLink stats={stats} address={address}/>
    </div>
}
