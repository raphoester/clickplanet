import {useEffect, useMemo, useState} from "react";
import {Countries, Country} from "../domain/countries.ts";
import {Allegiance, Lock, lockOf} from "../domain/allegiance.ts";
import {describeEnd} from "./endTime.ts";
import {AllegianceLockedError} from "../backends/backend.ts";
import SelectWithSearch from "./components/SelectWithSearch.tsx";
import {useStanding} from "./knockout/standingStore.ts";
import {stands} from "../domain/standing.ts";
import {Key} from "../i18n/translator.ts";
import {t, translator} from "../i18n/page.ts";
import "./CountryPicker.css"

export type CountryPickerProps = {
    country: Country,
    setCountry: (country: Country) => Promise<void> | void,
    allegiance?: Allegiance,
}

export const LOCKED_LINE: Key = "picker.lockedNow"

function ruleLineOf(lock: Lock | undefined, now: number): string {
    if (lock === undefined) return t("picker.onceAnHour")
    switch (lock.kind) {
        case "rules":
            return t(LOCKED_LINE)
        case "wait":
            return t("picker.onceAnHourNext", {time: describeEnd(lock.until, now)})
    }
}

export default function CountryPicker(props: CountryPickerProps) {
    const now = useNowUntilUnlocked(props.allegiance)
    const standing = useStanding((store) => store.standing)
    const locked = lockOf(props.allegiance, now, standing)
    const values = useMemo(
        () => Array.from(Countries.keys())
            .filter((code) => stands(standing, code))
            .map((code) => ({code, name: translator().country(code)}))
            .sort((a, b) => translator().compare(a.name, b.name)),
        [standing],
    )
    const [sending, setSending] = useState(false)
    const [failed, setFailed] = useState(false)

    const pick = async (country: Country) => {
        if (sending) return
        setSending(true)
        setFailed(false)
        try {
            await props.setCountry(country)
        } catch (e) {
            if (!(e instanceof AllegianceLockedError)) {
                console.error("could not change country", e)
                setFailed(true)
            }
        } finally {
            setSending(false)
        }
    }

    return <>
        <p className="country-picker-rule" role="status">{ruleLineOf(locked, now)}</p>
        {failed && <p className="country-picker-failed" role="alert">{t("picker.failed")}</p>}
        {!locked && <div className="country-picker-list" aria-busy={sending || undefined}>
            <SelectWithSearch
                onChange={(value) => pick(Countries.get(value.code) ?? {code: value.code})}
                selected={values.find((value) => value.code === props.country.code)}
                values={values}
            />
        </div>}
    </>
}

function useNowUntilUnlocked(allegiance: Allegiance | undefined): number {
    const [, setUnlocked] = useState(0)

    useEffect(() => {
        const left = (allegiance?.changeableAt ?? 0) - Date.now()
        if (left <= 0) return
        const timer = setTimeout(() => setUnlocked((n) => n + 1), left)
        return () => clearTimeout(timer)
    }, [allegiance])

    return Date.now()
}
