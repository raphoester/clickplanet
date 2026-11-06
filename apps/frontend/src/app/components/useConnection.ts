import {useEffect, useState} from 'react'
import {Connection, ConnectionSource} from '../../backends/connection.ts'

export function useConnection(source: ConnectionSource): Connection {
    const [connection, setConnection] = useState<Connection>("up")

    useEffect(() => source.watchConnection(setConnection), [source])

    return connection
}
