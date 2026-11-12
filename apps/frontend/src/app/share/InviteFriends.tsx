import {InviteLink} from "../../backends/crazyGames.ts";
import CountryFlag from "../components/CountryFlag.tsx";
import InviteButton from "./InviteButton.tsx";
import {useCountry} from "../countryStore.ts";
import {t, translator} from "../../i18n/page.ts";
import "./RecruitLink.css"

export type InviteFriendsProps = {
    invite: InviteLink
}

export default function InviteFriends({invite}: InviteFriendsProps) {
    const country = useCountry((store) => store.country)
    return <div className="panel-box recruit">
        <span className="recruit-title">
            <CountryFlag code={country.code}/>
            <span className="recruit-country">{t("recruit.title", {country: translator().country(country.code)})}</span>
        </span>
        <InviteButton flag={country.code}
                      invite={invite}
                      className="button button-secondary recruit-send"
                      label={t("invite.friends")}/>
    </div>
}
