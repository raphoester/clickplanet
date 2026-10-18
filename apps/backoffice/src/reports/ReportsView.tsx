import {useState} from "react"
import {Report} from "../gen/grpc/support/v1/admin_pb.ts"
import {ReportDetail} from "./ReportDetail.tsx"
import {ReportList} from "./ReportList.tsx"

// The list is read again on the way back: a report closed in the detail leaves the open list.
export function ReportsView() {
    const [openOnly, setOpenOnly] = useState(true)
    const [opened, setOpened] = useState<Report | undefined>(undefined)

    if (opened !== undefined) return <ReportDetail report={opened} onBack={() => setOpened(undefined)}/>
    return <ReportList openOnly={openOnly} onOpenOnly={setOpenOnly} onOpen={setOpened}/>
}
