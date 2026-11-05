import {Country} from "../../domain/countries.ts";
import {InviteLink} from "../../backends/crazyGames.ts";
import CountryFlag from "../components/CountryFlag.tsx";
import InviteButton from "./InviteButton.tsx";
import "./RecruitLink.css"

export type InviteFriendsProps = {
    country: Country
    invite: InviteLink
}

export default function InviteFriends({country, invite}: InviteFriendsProps) {
    return <div className="panel-box recruit">
        <span className="recruit-title">
            <CountryFlag code={country.code}/>
            <span className="recruit-country">Recruit for {country.name}</span>
        </span>
        <InviteButton flag={country.code}
                      invite={invite}
                      className="button button-secondary recruit-send"
                      label="Invite friends"/>
    </div>
}
