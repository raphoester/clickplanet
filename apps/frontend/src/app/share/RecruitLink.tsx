import {ShareAddress, shareHeadline, shareLabel, ShareStats} from "../../domain/shareCard.ts";
import CountryFlag from "../components/CountryFlag.tsx";
import RecruitButton from "./RecruitButton.tsx";
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
            <span className="recruit-country">Recruit for {stats.country.name}</span>
        </span>
        <span className="recruit-link">{shareLabel(address)}</span>
        <RecruitButton address={address}
                       text={shareHeadline(stats)}
                       className="button button-secondary recruit-send"/>
    </div>
}
