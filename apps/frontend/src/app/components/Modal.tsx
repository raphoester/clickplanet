import {ReactNode, useId, useRef} from "react";
import {createPortal} from "react-dom";
import {CloseIcon} from "./icons.tsx";
import {useModalDialog} from "./useDialog.ts";
import "./Modal.css"

export type ModalProps = {
    title: ReactNode;
    children: ReactNode;
    footer?: ReactNode;
    stayOnBackdropClick?: boolean;
    className?: string;
    onClose: () => void;
}

export default function Modal(props: ModalProps) {
    const panel = useRef<HTMLDivElement>(null)
    const titleId = useId()

    useModalDialog(panel, props.onClose)

    return createPortal(
        // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions
        <div
            className="modal"
            onClick={(e) => {
                if (props.stayOnBackdropClick) return
                if (e.target === e.currentTarget) props.onClose()
            }}>
            <div
                ref={panel}
                className={props.className ? `modal-content panel ${props.className}` : "modal-content panel"}
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
        </div>,
        document.body,
    )
}
