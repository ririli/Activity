package application

import domain "activity/services/activity-service/internal/participation/domain"

type Service struct{ Repo Repository }

func (s Service) ApplyDraw(p *domain.Progress, flowers int, value, cost, now int64) error {
	next := *p
	if err := next.ApplyDraw(flowers, value, cost, now); err != nil {
		return err
	}
	if err := s.Repo.SaveDrawProgress(next); err != nil {
		return err
	}
	*p = next
	return nil
}
func (s Service) Claim(p *domain.Progress, stage int) error {
	next := *p
	if err := next.Claim(stage); err != nil {
		return err
	}
	if err := s.Repo.UpdateRoundMask(next.Mask); err != nil {
		return err
	}
	*p = next
	return nil
}
