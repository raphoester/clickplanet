import {ReactNode, useId, useRef} from "react";
import CloseButton from "./CloseButton.tsx";
import {useModalDialog} from "./useDialog.ts";
import "./Modal.css"

export type ModalProps = {
    title: string;
    children: ReactNode;
    onClose: () => void;
}

/**
 * A true modal: a backdrop that covers the page and swallows the click, with a
 * centred panel above it. It says so to assistive tech, and holds the keyboard
 * while it is open — Escape closes, and Tab cannot wander onto the globe or the
 * menu still sitting behind the backdrop.
 *
 * The menu's panels deliberately do not use this. They sit inside the menu card
 * and block nothing, which is what makes them a menu rather than a dialog — see
 * MenuPanel.
 */
export default function Modal(props: ModalProps) {
    const panel = useRef<HTMLDivElement>(null)
    const titleId = useId()

    useModalDialog(panel, props.onClose)

    /*
     * Closing on the backdrop is a mouse convenience, so there is no keyboard
     * binding to add here: Escape and the close button are the keyboard paths,
     * and both are covered. Reading the event target beats stopping propagation
     * on the panel, which left the dialog carrying a click handler of its own.
     */
    return (
        // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions
        <div
            className="modal"
            onClick={(e) => {
                if (e.target === e.currentTarget) props.onClose()
            }}>
            <div
                ref={panel}
                className="modal-content"
                role="dialog"
                aria-modal="true"
                aria-labelledby={titleId}
                tabIndex={-1}>
                <div className="modal-header">
                    <h2 id={titleId}>{props.title}</h2>
                </div>
                {props.children}
                <CloseButton onClick={props.onClose}/>
            </div>
        </div>
    )
}
