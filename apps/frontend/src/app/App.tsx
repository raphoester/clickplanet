import PlayScreen, {PlayScreenProps} from "./play/PlayScreen.tsx";
import {WordFilter} from "../domain/wordFilter.ts";
import {WordFilterContext} from "./crazygames/wordFilter.ts";

export type AppProps = PlayScreenProps & {
    words: WordFilter
}

export default function App({words, ...play}: AppProps) {
    return <WordFilterContext.Provider value={words}>
        <PlayScreen {...play}/>
    </WordFilterContext.Provider>
}
