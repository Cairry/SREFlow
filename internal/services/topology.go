package services

import (
	"RCAgent/internal/ctx"
	"RCAgent/internal/models"
	"RCAgent/internal/types"
	"RCAgent/pkg/tools"
	"time"
)

type (
	topologyService struct {
		ctx *ctx.Context
	}

	TopologyService interface {
		Create(req interface{}) (data interface{}, err error)
		Update(req interface{}) (data interface{}, err error)
		Delete(req interface{}) (data interface{}, err error)
		List(req interface{}) (data interface{}, err error)
		GetDetail(req interface{}) (data interface{}, err error)
	}
)

func newTopologyService(ctx *ctx.Context) TopologyService {
	return &topologyService{
		ctx: ctx,
	}
}

func (t topologyService) Create(req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestTopologyCreate)
	topology := models.Topology{
		TenantId:  r.TenantId,
		ID:        "topo-" + tools.RandId(),
		Name:      r.Name,
		Nodes:     r.Nodes,
		Edges:     r.Edges,
		UpdatedBy: r.UpdatedBy,
		UpdatedAt: time.Now().Unix(),
	}

	err = t.ctx.Database.Topology().Create(topology)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (t topologyService) Update(req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestTopologyUpdate)
	topology := models.Topology{
		TenantId:  r.TenantId,
		ID:        r.ID,
		Name:      r.Name,
		Nodes:     r.Nodes,
		Edges:     r.Edges,
		UpdatedBy: r.UpdatedBy,
		UpdatedAt: time.Now().Unix(),
	}

	err = t.ctx.Database.Topology().Update(topology)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (t topologyService) Delete(req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestTopologyDelete)
	err = t.ctx.Database.Topology().Delete(r.TenantId, r.ID)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (t topologyService) List(req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestTopologyQuery)
	list, count, err := t.ctx.Database.Topology().List(r.TenantId, r.Query, r.Page)
	if err != nil {
		return nil, err
	}

	return types.ResponseTopologyList{
		List: list,
		Page: models.Page{
			Total: count,
			Index: r.Page.Index,
			Size:  r.Page.Size,
		},
	}, nil
}

// GetDetail 获取拓扑的完整信息，包括nodes和edges
func (t topologyService) GetDetail(req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestTopologyQuery)
	data, err = t.ctx.Database.Topology().GetDetail(r.TenantId, r.ID)
	if err != nil {
		return nil, err
	}
	return data, nil
}
