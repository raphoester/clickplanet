import {useEffect} from 'react'
import {QuizMaster} from '../../backends/backend.ts'
import {PlaySound} from '../sound/soundPlayer.ts'
import {useQuiz} from './quizStore.ts'

export function useQuizMaster(master: QuizMaster, countryCode: string, play: PlaySound) {
    useEffect(() => {
        useQuiz.setState({link: {master, countryCode, play}})
        return () => useQuiz.setState({link: undefined})
    }, [master, countryCode, play])

    useEffect(() => master.listenForQuizzes(useQuiz.getState().offer), [master])
    useEffect(() => master.listenForRewards(useQuiz.getState().rewarded), [master])
}
