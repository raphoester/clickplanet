import {Allegiance} from "../../domain/allegiance.ts"
import {Country} from "../../domain/countries.ts"
import {t, translator} from "../../i18n/page.ts"
import {PickAnotherWindow} from "../knockout/OutModal.tsx"
import {useStanding} from "../knockout/standingStore.ts"

export type ClosedFlagModalProps = {
    allegiance?: Allegiance
    onPick: (country: Country) => Promise<void>
}

// The flag held is closed: no click plays for it, so the window stays until the player picks another.
export default function ClosedFlagModal({allegiance, onPick}: ClosedFlagModalProps) {
    const standing = useStanding((store) => store.standing)
    if (!allegiance?.closed) return null
    return <PickAnotherWindow title={t("flagClosed.title", {country: translator().country(allegiance.countryId)})}
                              line={t("flagClosed.pickAnother")}
                              standing={standing}
                              allegiance={allegiance}
                              onPick={onPick}/>
}
