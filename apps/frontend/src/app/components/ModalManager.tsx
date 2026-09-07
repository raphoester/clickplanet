import {ReactNode, useState} from "react";
import Modal from "./Modal.tsx";
import BlockButton, {BlockButtonProps} from "./BlockButton.tsx";

type ModalManagerProps = {
    openByDefault?: boolean;
    modalTitle: string;
    children: ReactNode;
    buttonProps: BlockButtonProps;
    closeButtonText?: string;
}

/** A button that opens a modal, and the modal it opens. */
export default function ModalManager(props: ModalManagerProps) {
    const [isOpen, setIsOpen] = useState(props.openByDefault ?? false)

    return <>
        <BlockButton {...props.buttonProps} onClick={() => setIsOpen(true)}/>

        {isOpen && <Modal
            title={props.modalTitle}
            onClose={() => setIsOpen(false)}
            closeButtonText={props.closeButtonText}
        >
            {props.children}
        </Modal>}
    </>
}
