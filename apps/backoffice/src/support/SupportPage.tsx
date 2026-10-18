import {useState} from "react"
import {NoteList} from "./NoteList.tsx"
import {ReportList} from "./ReportList.tsx"

type View = "notes" | "reports"

export function SupportPage() {
    const [view, setView] = useState<View>("notes")

    return (
        <>
            <div className="segmented segmented-wide" role="group" aria-label="Support">
                <button type="button" aria-pressed={view === "notes"} onClick={() => setView("notes")}>Notes</button>
                <button type="button" aria-pressed={view === "reports"} onClick={() => setView("reports")}>Reports</button>
            </div>
            {view === "notes" ? <NoteList/> : <ReportList/>}
        </>
    )
}
