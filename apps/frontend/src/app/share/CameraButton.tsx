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
 *
 * **The label never changes**, and on a phone it is not drawn at all. A pill
 * anchored to a corner that rewrites its own label resizes under the cursor, and
 * there is nothing for the words to say here anyway: what answers the press is
 * the preview opening. Working is said by the button dimming instead.
 */
export default function CameraButton({busy, onClick}: CameraButtonProps) {
    return <button type="button"
                   className="camera-button"
                   // The label is gone under 768px, so the name lives here.
                   aria-label="Take a picture of your planet"
                   aria-busy={busy}
                   disabled={busy}
                   onClick={onClick}>
        <CameraIcon/>
        <span className="camera-button-label">Take a picture</span>
    </button>
}
