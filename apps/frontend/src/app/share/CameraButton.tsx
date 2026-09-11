import {CameraIcon} from "../components/icons.tsx";
import "./CameraButton.css"

export type CameraButtonProps = {
    busy?: boolean
    onClick: () => void
}

/**
 * Takes a picture of the globe as it is framed.
 *
 * It sits on the canvas rather than in the menu because that is what it is
 * about: the globe is the subject, and the card over it is not in the picture.
 * The menu's own row of actions is the wrong shape for it too — those are the
 * things you do with the *page*, and they are 56px slabs.
 */
export default function CameraButton({busy, onClick}: CameraButtonProps) {
    return <button type="button"
                   className="camera-button"
                   aria-busy={busy}
                   disabled={busy}
                   onClick={onClick}>
        <CameraIcon/>
        <span aria-live="polite">{busy ? "One sec…" : "Photo"}</span>
    </button>
}
