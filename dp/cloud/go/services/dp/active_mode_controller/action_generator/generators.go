package action_generator

import (
	"magma/dp/cloud/go/services/dp/active_mode_controller/action_generator/action"
	"magma/dp/cloud/go/services/dp/active_mode_controller/action_generator/sas"
	"magma/dp/cloud/go/services/dp/active_mode_controller/action_generator/sas/eirp"
	"magma/dp/cloud/go/services/dp/active_mode_controller/action_generator/sas/grant"
	"magma/dp/cloud/go/services/dp/storage"
	"magma/dp/cloud/go/services/dp/storage/db"
)

type actionGeneratorPerCbsd interface {
	generateActions(*storage.DetailedCbsd) []action.Action
}

type nothingGenerator struct{}

func (*nothingGenerator) generateActions(_ *storage.DetailedCbsd) []action.Action {
	return nil
}

type sasRequestGenerator struct {
	g sasGenerator
}

type sasGenerator interface {
	GenerateRequests(*storage.DetailedCbsd) []*storage.MutableRequest
}

func (s *sasRequestGenerator) generateActions(cbsd *storage.DetailedCbsd) []action.Action {
	actions := grant.RemoveIdleGrants(cbsd)
	reqs := s.g.GenerateRequests(cbsd)
	for _, r := range reqs {
		if r != nil {
			r.Request.CbsdId = cbsd.Cbsd.Id
			actions = append(actions, &action.Request{Data: r})
		}
	}
	return actions
}

type deleteGenerator struct{}

func (*deleteGenerator) generateActions(cbsd *storage.DetailedCbsd) []action.Action {
	act := &action.DeleteCbsd{Id: cbsd.Cbsd.Id.Int64}
	return []action.Action{act}
}

type acknowledgeDeregisterGenerator struct{}

func (a *acknowledgeDeregisterGenerator) generateActions(cbsd *storage.DetailedCbsd) []action.Action {
	data := &storage.DBCbsd{
		Id:               cbsd.Cbsd.Id,
		ShouldDeregister: db.MakeBool(false),
	}
	mask := db.NewIncludeMask("should_deregister")
	act := &action.UpdateCbsd{Data: data, Mask: mask}
	return []action.Action{act}
}

type acknowledgeRelinquishGenerator struct{}

func (a *acknowledgeRelinquishGenerator) generateActions(cbsd *storage.DetailedCbsd) []action.Action {
	data := &storage.DBCbsd{
		Id:               cbsd.Cbsd.Id,
		ShouldRelinquish: db.MakeBool(false),
	}
	mask := db.NewIncludeMask("should_relinquish")
	act := &action.UpdateCbsd{Data: data, Mask: mask}
	return []action.Action{act}
}

type storeAvailableFrequenciesGenerator struct{}

func (s *storeAvailableFrequenciesGenerator) generateActions(cbsd *storage.DetailedCbsd) []action.Action {
	calc := eirp.NewCalculator(cbsd.Cbsd)
	frequencies := grant.CalcAvailableFrequencies(cbsd.Cbsd.Channels, calc)
	data := &storage.DBCbsd{
		Id:                   cbsd.Cbsd.Id,
		AvailableFrequencies: frequencies,
	}
	mask := db.NewIncludeMask("available_frequencies")
	act := &action.UpdateCbsd{Data: data, Mask: mask}
	return []action.Action{act}
}

type grantManager struct {
	nextSendTimestamp int64
	rng               RNG
}

func (g *grantManager) GenerateRequests(cbsd *storage.DetailedCbsd) []*storage.MutableRequest {
	grants := grant.GetFrequencyGrantMapping(cbsd.Grants)
	calc := eirp.NewCalculator(cbsd.Cbsd)
	processors := grant.Processors[*storage.MutableRequest]{
		Del: &sas.RelinquishmentProcessor{
			CbsdId: cbsd.Cbsd.CbsdId.String,
			Grants: grants,
		},
		Keep: &sas.HeartbeatProcessor{
			NextSendTimestamp: g.nextSendTimestamp,
			CbsdId:            cbsd.Cbsd.CbsdId.String,
			Grants:            grants,
		},
		Add: &sas.GrantProcessor{
			CbsdId:   cbsd.Cbsd.CbsdId.String,
			Calc:     calc,
			Channels: cbsd.Cbsd.Channels,
		},
	}
	dbGrants := make([]*storage.DBGrant, len(cbsd.Grants))
	for i, gt := range cbsd.Grants {
		dbGrants[i] = gt.Grant
	}
	requests := grant.ProcessGrants(cbsd.Cbsd, dbGrants, processors, g.rng.Int())
	if len(requests) > 0 {
		return requests
	}
	gen := sas.SpectrumInquiryRequestGenerator{}
	return gen.GenerateRequests(cbsd)
}
