package application

func nextRound(s *work, from int) (any, error) {
	if from > 0 && s.progress.Round == from+1 {
		return s.Summary(), nil
	}
	pending, err := s.session.Participation.PendingStageChoices(s.progress.Round)
	if err != nil {
		return nil, err
	}
	if err = s.progress.CheckNext(from, pending); err != nil {
		return nil, err
	}
	if err = s.session.Participation.CloseRound(s.progress.Round); err != nil {
		return nil, err
	}
	if err = s.session.Participation.InsertRound(s.progress.Round+1, s.progress.Version); err != nil {
		return nil, err
	}
	if err = s.session.Participation.IncrementParticipationRound(); err != nil {
		return nil, err
	}
	return map[string]int{"round_no": s.progress.Round + 1}, nil
}
