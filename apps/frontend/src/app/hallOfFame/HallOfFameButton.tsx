import {Trophy} from "../season/SeasonFacts.tsx"
import "../season/Season.css"
import "./HallOfFame.css"

export type HallOfFameButtonProps = {
    compact: boolean
    besideSeason: boolean
    onOpen: () => void
}

export default function HallOfFameButton({compact, besideSeason, onOpen}: HallOfFameButtonProps) {
    const className = compact
        ? "panel-box hall-of-fame-button hall-of-fame-button--compact"
        : besideSeason ? "panel hall-of-fame-button hall-of-fame-button--beside" : "panel hall-of-fame-button"

    return <button type="button"
                   className={className}
                   aria-label="Hall of Fame"
                   aria-haspopup="dialog"
                   title="Hall of Fame"
                   onClick={onOpen}>
        <Trophy/>
    </button>
}
