import {ShareAddress, shareLabel, ShareStats} from "../../domain/shareCard.ts";
import {t, translator} from "../../i18n/page.ts";
import CountryFlag from "../components/CountryFlag.tsx";
import RecruitButton from "./RecruitButton.tsx";
import {shareHeadline} from "./shareLines.ts";
import "./RecruitLink.css"

export type RecruitLinkProps = {
    stats: ShareStats
    address: ShareAddress
}

export default function RecruitLink({stats, address}: RecruitLinkProps) {
    const code = stats.country.code

    return <div className="panel-box recruit">
        <span className="recruit-title">
            <CountryFlag code={code}/>
            <span className="recruit-country">{t("recruit.title", {country: translator().country(code)})}</span>
        </span>
        <span className="recruit-link">{shareLabel(address)}</span>
        <RecruitButton address={address}
                       text={shareHeadline(stats)}
                       className="button button-secondary recruit-send"/>
    </div>
}
