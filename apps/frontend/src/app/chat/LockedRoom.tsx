import {Country} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {LockIcon} from "../components/icons.tsx"
import {newMessages} from "./newMessages.ts"

export type LockedRoomProps = {
    country: Country
    unread: number
    onSignIn: () => void
}

const GHOST_WIDTHS = [58, 36, 72, 44, 64, 30, 52, 68, 40, 60, 34, 70, 48, 56]

export default function LockedRoom({country, unread, onSignIn}: LockedRoomProps) {
    return <div className="chat-locked">
        <div className="chat-locked-ghosts" aria-hidden="true">
            {GHOST_WIDTHS.map((width, i) => <span key={i} className="chat-locked-ghost" style={{width: `${width}%`}}/>)}
        </div>

        <div className="chat-locked-card panel-box">
            <span className="chat-locked-lock"><LockIcon size={22}/></span>
            <p className="chat-locked-title"><CountryFlag code={country.code}/> {country.name} team chat</p>
            {unread > 0 && <p className="chat-locked-count">{newMessages(unread)}</p>}
            <button type="button" className="button button-action" onClick={onSignIn}>Sign in to join</button>
        </div>
    </div>
}
