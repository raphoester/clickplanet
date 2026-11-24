import "../support/Support.css"
import "./FlagSubmission.css"

// The operator's own words to the founder, quoted: a rejection's note or an early close's.
export default function FlagNote({text}: {text: string}) {
    return <blockquote className="support-quoted flag-status-note">
        <span className="support-quoted-text">{text}</span>
    </blockquote>
}
