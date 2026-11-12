import Modal from "../components/Modal.tsx";
import {ShareAddress, ShareStats} from "../../domain/shareCard.ts";
import {t} from "../../i18n/page.ts";
import ShareActions from "./ShareActions.tsx";
import {Shot} from "./useSharePicture.ts";
import {pictureAlt, shareText} from "./shareLines.ts";
import "./SharePreview.css"

export type SharePreviewProps = {
    shot: Shot
    stats: ShareStats
    address: ShareAddress
    onClose: () => void
}

export default function SharePreview({shot, stats, address, onClose}: SharePreviewProps) {
    return <Modal title={t("share.pictureTitle")}
                  className="modal-picture"
                  footer={shot.kind === "ready"
                      ? <ShareActions file={shot.file} text={shareText(stats, address)}/>
                      : undefined}
                  onClose={onClose}>
        {shot.kind === "ready"
            ? <img className="share-picture"
                   src={shot.url}
                   alt={pictureAlt(stats)}/>
            : <p className="share-picture-failed">{t("share.pictureFailed")}</p>}
    </Modal>
}
