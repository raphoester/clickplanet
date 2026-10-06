import Viewer, {ViewerProps} from "./viewer/Viewer.tsx";
import DonationModal from "./components/DonationModal.tsx";

export type AppProps = ViewerProps

export default function App(props: AppProps) {
    return <>
        <DonationModal/>

        <Viewer
            sharedCountry={props.sharedCountry}
            tileClicker={props.tileClicker}
            ownershipsGetter={props.ownershipsGetter}
            updatesListener={props.updatesListener}
            clickBudgetSource={props.clickBudgetSource}
            bonusListener={props.bonusListener}
            quizMaster={props.quizMaster}
            bomber={props.bomber}
            refiller={props.refiller}
            garrisons={props.garrisons}
            chatBackend={props.chatBackend}
            account={props.account}
            presence={props.presence}
            playerInfo={props.playerInfo}
            season={props.season}
            standings={props.standings}
        />
    </>
}
