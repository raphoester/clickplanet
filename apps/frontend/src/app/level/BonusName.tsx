import {ChargeKind} from "../../domain/bonus.ts"
import BonusIcon from "../components/BonusIcon.tsx"
import {kindName} from "../bonus/kindName.ts"
import "./Level.css"

// A kind of bonus inside a line: its box in its color, and its name.
export default function BonusName({kind}: {kind: ChargeKind}) {
    return <span className={`bonus-name bonus-name--${kind}`}>
        <span className="bonus-name-box" aria-hidden="true"><BonusIcon kind={kind}/></span>
        <span className="bonus-name-text">{kindName(kind)}</span>
    </span>
}
