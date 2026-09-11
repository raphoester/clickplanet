import Viewer, {ViewerProps} from "./viewer/Viewer.tsx";
import DonationModal from "./components/DonationModal.tsx";

export type AppProps = ViewerProps

export default function App(props: AppProps) {
    return <>
        <DonationModal/>

        <Viewer
            tileClicker={props.tileClicker}
            ownershipsGetter={props.ownershipsGetter}
            updatesListener={props.updatesListener}
            clickBudgetSource={props.clickBudgetSource}
            chatBackend={props.chatBackend}
        />
    </>
}
