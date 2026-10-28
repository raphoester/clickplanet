import {useEffect, useState} from "react"
import {ClipsNameBackend} from "../../backends/clips.ts"
import Switch from "../components/Switch.tsx"

export default function ClipsSwitch({clips}: {clips: ClipsNameBackend}) {
    const [shown, setShown] = useState<boolean>()
    const [saving, setSaving] = useState(false)
    const [failed, setFailed] = useState(false)

    useEffect(() => {
        let live = true
        clips.nameShown().then(
            (read) => {
                if (live) setShown(read)
            },
            (e) => console.error("Could not read whether the clips show your name", e),
        )
        return () => {
            live = false
        }
    }, [clips])

    if (shown === undefined) return null

    const save = async (on: boolean) => {
        setShown(on)
        setSaving(true)
        setFailed(false)
        try {
            setShown(await clips.setNameShown(on))
        } catch {
            setShown(!on)
            setFailed(true)
        } finally {
            setSaving(false)
        }
    }

    return <div className="account-clips" aria-busy={saving}>
        <div className="panel-box">
            <Switch label="Show my name in clips" checked={shown} disabled={saving} onChange={(on) => void save(on)}/>
        </div>
        {failed && <p className="account-failure" role="alert">Something went wrong. Try again.</p>}
    </div>
}
