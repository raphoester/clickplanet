import {Season} from "../../backends/season.ts"
import {finaleCalendar} from "../../domain/seasonCalendar.ts"
import {download} from "../download.ts"
import {CalendarIcon} from "../components/icons.tsx"
import "./Season.css"

export default function AddToCalendarButton({season}: {season: Season}) {
    const add = () => {
        const calendar = finaleCalendar(season, new URL("/play", window.location.origin).href, Date.now())
        download(new File([calendar.text], calendar.name, {type: "text/calendar"}))
    }

    return <button type="button" className="button button-mini season-calendar" onClick={add}>
        <CalendarIcon/>
        <span>Add to calendar</span>
    </button>
}
