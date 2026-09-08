import {ReactNode, useId, useRef} from "react";
import {CloseIcon} from "./icons.tsx";
import {useModalDialog} from "./useDialog.ts";
import "./Modal.css"

export type ModalProps = {
    title: string;
    children: ReactNode;
    /**
     * Pinned below the scrolling body. A dialog's primary action does not
     * belong in the scroll: "Buy me a coffee" used to sit at the foot of the
     * About copy, where a short screen showed nothing of it but an orange line.
     */
    footer?: ReactNode;
    onClose: () => void;
}

/**
 * A true modal: a backdrop that covers the page and swallows the click, with a
 * centred panel above it. It says so to assistive tech, and holds the keyboard
 * while it is open — Escape closes, and Tab cannot wander onto the globe or the
 * menu still sitting behind the backdrop.
 *
 * It closes on the × beside its title. That is the whole reason a dialog is the
 * right shell for something like About: it opened over everything, so an × says
 * exactly what will happen, where a "Back" or a "Close" at the bottom of a
 * panel had to compete with whatever the card's own controls meant.
 *
 * The menu's country picker deliberately does not use this. It sits inside the
 * menu card and blocks nothing, which is what makes it a menu rather than a
 * dialog — see MenuPanel.
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
                    <button type="button"
                            className="icon-button"
                            aria-label="Close"
                            onClick={props.onClose}>
                        <CloseIcon/>
                    </button>
                </div>
                <div className="modal-scroll">
                    <div className="modal-body">
                        {props.children}
                    </div>
                </div>
                {props.footer && <div className="modal-footer">{props.footer}</div>}
            </div>
        </div>
    )
}
