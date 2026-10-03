import {CameraIcon} from "../components/icons.tsx";
import "./CameraButton.css"

export type CameraButtonProps = {
    busy?: boolean
    onClick: () => void
}

export default function CameraButton({busy, onClick}: CameraButtonProps) {
    return <button type="button"
                   className="button camera-button"
                   aria-label="Take a picture of your planet"
                   aria-busy={busy}
                   disabled={busy}
                   onClick={onClick}>
        <CameraIcon/>
        <span className="camera-button-label">Take a picture</span>
    </button>
}
