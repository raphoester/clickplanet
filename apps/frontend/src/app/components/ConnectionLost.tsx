import './ConnectionLost.css'

export type ConnectionLostProps = {
    lowered?: boolean
}

export default function ConnectionLost({lowered}: ConnectionLostProps) {
    return <div className={`connection-lost${lowered ? " connection-lost--lowered" : ""}`} role="alert">
        <div className="connection-lost-line panel">
            <span aria-hidden="true">📡</span>
            <span><strong>Connection lost</strong> Check your network</span>
        </div>
    </div>
}
