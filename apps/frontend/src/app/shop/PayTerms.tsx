import {useId} from "react"
import {t} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"
import {LockIcon} from "../components/icons.tsx"
import "./Shop.css"

// When what is bought starts: a new flag once approved, a closed flag once paid, time added to an open flag's end.
export type Start = "approved" | "paid" | "added"

const CONSENTS: Record<Start, Key> = {
    approved: "shop.consent",
    paid: "shop.consentPaid",
    added: "shop.consentAdded",
}

export type ConsentProps = {
    starts: Start
    agreed: boolean
    disabled: boolean
    onAgree: (agreed: boolean) => void
}

// The EU's waiver of the 14 days to cancel: the server takes no checkout without it.
export function Consent({starts, agreed, disabled, onAgree}: ConsentProps) {
    const id = useId()
    return <label className="shop-consent" htmlFor={id}>
        <input id={id}
               type="checkbox"
               checked={agreed}
               disabled={disabled}
               onChange={(e) => onAgree(e.target.checked)}/>
        <span>{t(CONSENTS[starts])}</span>
    </label>
}

export function FinePrint({renews}: {renews: boolean}) {
    return <>
        <p className="shop-trust">
            <LockIcon size={13}/>
            <span>{renews ? t("shop.secureStopAnytime") : t("shop.secure")}</span>
        </p>
        <p className="shop-legal">
            <a href="/sale" target="_blank" rel="noopener noreferrer">{t("shop.sale")}</a>
            <a href="/refunds" target="_blank" rel="noopener noreferrer">{t("shop.refunds")}</a>
        </p>
    </>
}
