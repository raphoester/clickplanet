import {useEffect, useId, useMemo, useRef, useState} from "react"
import {t, translator} from "../../i18n/page.ts"
import {ANTHEMS} from "../anthem/anthemsAsset.ts"

type LibraryAnthem = {code: string, title: string, country: string}

// Every recording of the game's library once, under the country it was made for (Anguilla plays the United Kingdom's
// file: it is listed as the United Kingdom's), by that country's name in the page's language.
function libraryAnthems(): LibraryAnthem[] {
    return Object.entries(ANTHEMS)
        .filter(([code, {url}]) => url.startsWith(`/static/anthems/${code}-`))
        .map(([code, {title}]) => ({code, title, country: translator().country(code)}))
        .sort((a, b) => translator().compare(a.country, b.country))
}

export type AnthemPickerProps = {
    // The code of the recording picked; none for no anthem.
    value: string | undefined
    disabled?: boolean
    onChange: (code: string | undefined) => void
}

// A sold flag's anthem, picked from the game's own recordings and heard before it is picked: never a file of the
// player's, whose rights nobody could check.
export default function AnthemPicker({value, disabled, onChange}: AnthemPickerProps) {
    const id = useId()
    const hintId = useId()
    const library = useMemo(libraryAnthems, [])
    const [playing, setPlaying] = useState(false)
    const audio = useRef<HTMLAudioElement>()
    const url = value ? ANTHEMS[value]?.url : undefined

    // A new pick, or the picker gone, stops what was heard.
    useEffect(() => () => {
        audio.current?.pause()
        audio.current = undefined
        setPlaying(false)
    }, [url])

    const listen = () => {
        if (!url) return
        if (playing) {
            audio.current?.pause()
            setPlaying(false)
            return
        }
        const heard = audio.current ?? new Audio(url)
        audio.current = heard
        heard.onended = () => setPlaying(false)
        heard.play().then(() => setPlaying(true), (e: unknown) => console.error("could not play the anthem", e))
    }

    return <>
        <label className="menu-label" htmlFor={id}>{t("flagSubmission.anthem")}</label>
        <div className="flag-form-anthem">
            <select id={id}
                    className="field flag-form-anthem-pick"
                    value={value ?? ""}
                    disabled={disabled}
                    aria-describedby={hintId}
                    onChange={(e) => onChange(e.target.value === "" ? undefined : e.target.value)}>
                <option value="">{t("flagSubmission.noAnthem")}</option>
                {library.map(({code, title, country}) => <option key={code} value={code}>
                    {t("flagSubmission.anthemOption", {country, title})}
                </option>)}
            </select>
            <button type="button"
                    className="button button-mini button-secondary"
                    disabled={!url}
                    aria-pressed={playing}
                    onClick={listen}>
                {playing ? t("flagSubmission.stopAnthem") : t("flagSubmission.listenAnthem")}
            </button>
        </div>
        <p id={hintId} className="flag-form-hint">{t("flagSubmission.anthemHint")}</p>
    </>
}
