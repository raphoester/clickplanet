import {useId} from "react"
import {isLanguage, LANGUAGES, nativeNameOf} from "../../i18n/languages.ts"
import {t, translator} from "../../i18n/page.ts"
import {keepPickedLanguage} from "../../i18n/pickedLanguage.ts"

export type LanguagePickerProps = {
    // The page reads every line once: a new language is a new load.
    reload?: () => void
}

export default function LanguagePicker({reload = () => window.location.reload()}: LanguagePickerProps) {
    const id = useId()

    return <div className="settings-language">
        <label className="menu-label" htmlFor={id}>{t("settings.language")}</label>
        <select id={id}
                className="field"
                value={translator().language}
                onChange={(event) => {
                    const picked = event.target.value
                    if (!isLanguage(picked) || picked === translator().language) return
                    keepPickedLanguage(picked)
                    reload()
                }}>
            {LANGUAGES.map((language) => <option key={language} value={language} lang={language}>
                {nativeNameOf(language)}
            </option>)}
        </select>
    </div>
}
