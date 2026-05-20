package api

import (
	"SREFlow/internal/services"
	"SREFlow/internal/types"

	"github.com/gin-gonic/gin"
)

type topology struct{}

var TopologyController = new(topology)

func (topologyController topology) API(gin *gin.RouterGroup) {
	a := gin.Group("topology")
	a.Use()
	{
		a.POST("createTopology", topologyController.Create)
		a.POST("updateTopology", topologyController.Update)
		a.POST("deleteTopology", topologyController.Delete)
	}

	b := gin.Group("topology")
	b.Use()
	{
		b.GET("listTopology", topologyController.List)
		b.GET("getTopologyDetail", topologyController.GetDetail)
	}
}

func (topology topology) Create(ctx *gin.Context) {
	r := new(types.RequestTopologyCreate)
	BindJson(ctx, r)

	tid, _ := ctx.Get("TenantID")
	r.TenantId = tid.(string)

	Service(ctx, func() (interface{}, error) {
		return services.Topology.Create(r)
	})
}

func (topology topology) Update(ctx *gin.Context) {
	r := new(types.RequestTopologyUpdate)
	BindJson(ctx, r)

	tid, _ := ctx.Get("TenantID")
	r.TenantId = tid.(string)

	Service(ctx, func() (interface{}, error) {
		return services.Topology.Update(r)
	})
}

func (topology topology) Delete(ctx *gin.Context) {
	r := new(types.RequestTopologyDelete)
	BindJson(ctx, r)

	tid, _ := ctx.Get("TenantID")
	r.TenantId = tid.(string)

	Service(ctx, func() (interface{}, error) {
		return services.Topology.Delete(r)
	})
}

func (topology topology) List(ctx *gin.Context) {
	r := new(types.RequestTopologyQuery)
	BindQuery(ctx, r)

	tid, _ := ctx.Get("TenantID")
	r.TenantId = tid.(string)

	Service(ctx, func() (interface{}, error) {
		return services.Topology.List(r)
	})
}

func (topology topology) GetDetail(ctx *gin.Context) {
	r := new(types.RequestTopologyQuery)
	BindQuery(ctx, r)

	tid, _ := ctx.Get("TenantID")
	r.TenantId = tid.(string)

	Service(ctx, func() (interface{}, error) {
		return services.Topology.GetDetail(r)
	})
}
