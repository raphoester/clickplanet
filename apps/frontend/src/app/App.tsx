import Viewer, {ViewerProps} from "./viewer/Viewer.tsx";
import DonationModal from "./components/DonationModal.tsx";
import {ON_CRAZYGAMES} from "../platform.ts";
import {WordFilter} from "../domain/wordFilter.ts";
import {WordFilterContext} from "./crazygames/wordFilter.ts";

export type AppProps = ViewerProps & {
    words: WordFilter
}

export default function App(props: AppProps) {
    return <WordFilterContext.Provider value={props.words}>
        {!ON_CRAZYGAMES && <DonationModal/>}

        <Viewer
            sharedCountry={props.sharedCountry}
            tileClicker={props.tileClicker}
            ownershipsGetter={props.ownershipsGetter}
            updatesListener={props.updatesListener}
            clickBudgetSource={props.clickBudgetSource}
            connection={props.connection}
            bonusListener={props.bonusListener}
            quizMaster={props.quizMaster}
            bomber={props.bomber}
            refiller={props.refiller}
            shielder={props.shielder}
            allegiance={props.allegiance}
            chatBackend={props.chatBackend}
            account={props.account}
            presence={props.presence}
            playerInfo={props.playerInfo}
            tileHolders={props.tileHolders}
            landmasses={props.landmasses}
            turn={props.turn}
            standings={props.standings}
            support={props.support}
            missions={props.missions}
            chargesRefresher={props.chargesRefresher}
            clips={props.clips}
            crazyGames={props.crazyGames}
        />
    </WordFilterContext.Provider>
}
