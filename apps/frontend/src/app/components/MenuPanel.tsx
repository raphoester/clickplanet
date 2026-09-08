import {ReactNode} from "react";
import CloseButton from "./CloseButton.tsx";
import "./MenuPanel.css"

export type MenuPanelProps = {
    title: string;
    children: ReactNode;
    onClose: () => void;
    closeButtonText?: string;
}

/**
 * A panel that expands inside the menu card.
 *
 * Unlike Modal there is no backdrop and no layer of its own: the card simply
 * grows, and the globe behind it stays visible and clickable. The panel is
 * capped by the card's own max-height and scrolls internally rather than
 * pushing the card off the bottom of the viewport.
 */
export default function MenuPanel(props: MenuPanelProps) {
    return <div className="menu-panel">
        <h2 className="menu-panel-title">{props.title}</h2>
        <div className="menu-panel-body">
            {props.children}
        </div>
        <CloseButton onClick={props.onClose} text={props.closeButtonText}/>
    </div>
}
