import {useMemo, useState} from "react"
import {Countries, Country} from "../../domain/countries.ts"
import {stands} from "../../domain/standing.ts"
import {t, translator} from "../../i18n/page.ts"
import Modal from "../components/Modal.tsx"
import SelectWithSearch from "../components/SelectWithSearch.tsx"
import {useStanding} from "./standingStore.ts"
import "../CountryPicker.css"

export type OutModalProps = {
    // The flag played: the one held, or a newcomer's choice.
    flag: Country
    onPick: (country: Country) => Promise<void>
}

// While the knock-out plays and the flag played is not in: the flags still in, to pick one.
export default function OutModal({flag, onPick}: OutModalProps) {
    const standing = useStanding((store) => store.standing)
    const watching = useStanding((store) => store.watching)
    const watch = useStanding((store) => store.watch)
    const [sending, setSending] = useState(false)
    const [failed, setFailed] = useState(false)
    const stillIn = useMemo(
        () => Array.from(Countries.keys())
            .filter((code) => standing !== undefined && stands(standing, code))
            .map((code) => ({code, name: translator().country(code)}))
            .sort((a, b) => translator().compare(a.name, b.name)),
        [standing],
    )

    if (standing === undefined || stands(standing, flag.code) || watching === flag.code) return null

    const pick = async (country: Country) => {
        if (sending) return
        setSending(true)
        setFailed(false)
        try {
            await onPick(country)
        } catch (e) {
            console.error("could not pick a flag still in", e)
            setFailed(true)
        } finally {
            setSending(false)
        }
    }

    const country = translator().country(flag.code)
    return <Modal title={t(standing.out.has(flag.code) ? "knockout.isOut" : "knockout.noCapital", {country})}
                  onClose={() => watch(flag.code)}>
        <p className="country-picker-rule">{t("knockout.pickStillIn")}</p>
        {failed && <p className="country-picker-failed" role="alert">{t("knockout.pickFailed")}</p>}
        <div className="country-picker-list" aria-busy={sending || undefined}>
            <SelectWithSearch values={stillIn} onChange={(value) => void pick(Countries.get(value.code) ?? {code: value.code})}/>
        </div>
    </Modal>
}
