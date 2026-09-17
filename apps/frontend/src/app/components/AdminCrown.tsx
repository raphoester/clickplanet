import {CrownIcon} from "./icons.tsx"
import "./AdminCrown.css"

/** Beside the name of an admin of the game: in the chat, the roster and the player card. */
export default function AdminCrown({size = 14}: {size?: number}) {
    return <span className="admin-crown" role="img" aria-label="Admin" title="Admin">
        <CrownIcon size={size}/>
    </span>
}
