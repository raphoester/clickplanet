import {useEffect, useState} from 'react'
import {Connection, ConnectionSource} from '../../backends/connection.ts'

export function useConnection(source: ConnectionSource | undefined): Connection {
    const [connection, setConnection] = useState<Connection>("up")

    useEffect(() => {
        if (!source) return
        return source.watchConnection(setConnection)
    }, [source])

    return connection
}
