import {ReactNode} from "react";
import CloseButton from "./CloseButton.tsx";
import "./Modal.css"

export type ModalProps = {
    title: string;
    children: ReactNode;
    onClose: () => void;
}

/**
 * A true modal: a backdrop that covers the page and swallows the click, with a
 * centred panel above it.
 *
 * The menu's panels deliberately do not use this. They expand inside the menu
 * card and leave the globe visible and clickable, which is what makes them a
 * menu rather than a dialog — see MenuPanel.
 */
export default function Modal(props: ModalProps) {
    return (<div className="modal" onClick={props.onClose}>
        <div
            className="modal-content"
            onClick={(e) => {
                e.stopPropagation()
            }}>
            <div className="modal-header">
                <h2>{props.title}</h2>
            </div>
            {props.children}
            <CloseButton onClick={props.onClose}/>
        </div>
    </div>)
}
