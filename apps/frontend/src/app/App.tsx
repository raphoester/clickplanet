import PlayScreen, {PlayScreenProps} from "./play/PlayScreen.tsx";
import DonationModal from "./components/DonationModal.tsx";
import {ON_CRAZYGAMES} from "../platform.ts";
import {WordFilter} from "../domain/wordFilter.ts";
import {WordFilterContext} from "./crazygames/wordFilter.ts";

export type AppProps = PlayScreenProps & {
    words: WordFilter
}

export default function App({words, ...play}: AppProps) {
    return <WordFilterContext.Provider value={words}>
        {!ON_CRAZYGAMES && <DonationModal/>}

        <PlayScreen {...play}/>
    </WordFilterContext.Provider>
}
