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
