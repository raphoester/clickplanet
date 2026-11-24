import {useId} from "react"
import {t} from "../../i18n/page.ts"
import {LockIcon} from "../components/icons.tsx"
import "./Shop.css"

export type ConsentProps = {
    agreed: boolean
    disabled: boolean
    onAgree: (agreed: boolean) => void
}

// The EU's waiver of the 14 days to cancel: the server takes no checkout without it.
export function Consent({agreed, disabled, onAgree}: ConsentProps) {
    const id = useId()
    return <label className="shop-consent" htmlFor={id}>
        <input id={id}
               type="checkbox"
               checked={agreed}
               disabled={disabled}
               onChange={(e) => onAgree(e.target.checked)}/>
        <span>{t("shop.consent")}</span>
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
