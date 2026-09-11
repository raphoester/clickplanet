import Modal from "../components/Modal.tsx";
import {ShareStats, shareText, statsLine} from "../../domain/shareCard.ts";
import ShareActions from "./ShareActions.tsx";
import {Shot} from "./useSharePicture.ts";
import "./SharePreview.css"

export type SharePreviewProps = {
    shot: Shot
    stats: ShareStats
    onClose: () => void
}

/**
 * The picture, before anything is done with it.
 *
 * A preview rather than a straight-to-the-clipboard button because the framing
 * is the player's: the camera takes the globe at whatever angle and zoom they
 * left it, and the one thing they cannot check afterwards is whether that was
 * the shot they wanted. It is also where the choice of what to do with it goes,
 * which is a choice with nowhere else to live.
 */
export default function SharePreview({shot, stats, onClose}: SharePreviewProps) {
    return <Modal title="Your planet"
                  className="modal-picture"
                  footer={shot.kind === "ready"
                      ? <ShareActions file={shot.file} text={shareText(stats)}/>
                      : undefined}
                  onClose={onClose}>
        {shot.kind === "ready"
            ? <img className="share-picture"
                   src={shot.url}
                   alt={`${stats.country.name} on the globe as you have it framed — ${statsLine(stats).toLowerCase()}`}/>
            : <p className="share-picture-failed">
                The picture could not be drawn. Reload the page and try again — if it
                keeps happening, the browser may have dropped the globe’s graphics
                context.
            </p>}
    </Modal>
}
