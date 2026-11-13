import {rich} from '../../i18n/rich.tsx'
import './ConnectionLost.css'

export default function ConnectionLost() {
    return <div className="connection-lost" role="alert">
        <div className="connection-lost-line panel">
            <span aria-hidden="true">📡</span>
            <span>{rich("connection.lost", {strong: (words) => <strong>{words}</strong>})}</span>
        </div>
    </div>
}
