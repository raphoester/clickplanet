import {useState} from "react";
import Viewer, {ViewerProps} from "./viewer/Viewer.tsx";
import OnLoadModal from "./components/OnLoadModal.tsx";
import BuyMeACoffee from "./components/BuyMeACoffee.tsx";

export type AppProps = ViewerProps

export default function App(props: AppProps) {
    /**
     * Decided once, when the app mounts. This used to be a bare `Math.random()`
     * in the returned JSX, which is a side effect in render: React is free to
     * render a component more than once for a single commit, and the modal
     * would appear or vanish on any re-render.
     */
    const [showDonationModal] = useState(() => Math.random() > 0.5)

    return <>
        {showDonationModal && <OnLoadModal title="Dear earthlings">
            <div className="center-align">
                <img alt="ClickPlanet logo"
                     src="/static/logo.svg"
                     width="64px"
                     height="auto"/>
            </div>
            <div className="modal-onload-text">
                <h3>Do you like ClickPlanet ?</h3>
                <p>It's free and open-source 🤗</p>
                <p>Sadly, the servers are expensive to run 😭</p>
                <p>Every contribution helps us keep this awesome platform running!</p>
            </div>
            <BuyMeACoffee/>
        </OnLoadModal>}

        <Viewer
            tileClicker={props.tileClicker}
            ownershipsGetter={props.ownershipsGetter}
            updatesListener={props.updatesListener}
        />
    </>
}
