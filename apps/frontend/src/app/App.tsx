import Viewer, {ViewerProps} from "./viewer/Viewer.tsx";
import DonationModal from "./components/DonationModal.tsx";
import {ON_CRAZYGAMES} from "../platform.ts";
import {WordFilter} from "../domain/wordFilter.ts";
import {WordFilterContext} from "./crazygames/wordFilter.ts";

export type AppProps = ViewerProps & {
    words: WordFilter
}

export default function App({words, ...viewer}: AppProps) {
    return <WordFilterContext.Provider value={words}>
        {!ON_CRAZYGAMES && <DonationModal/>}

        <Viewer {...viewer}/>
    </WordFilterContext.Provider>
}
