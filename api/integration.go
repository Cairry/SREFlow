package api

import (
	"SREFlow/internal/services"
	"SREFlow/internal/types"
	"net/http"

	"github.com/gin-gonic/gin"
)

type integration struct{}

var IntegrationController = new(integration)

func (integration) API(gin *gin.RouterGroup) {
	d := gin.Group("integrations")
	{
		d.GET("", listIntegrations)
		d.GET("/:id", getIntegration)
		d.PUT("/:id", updateIntegration)
		d.GET("/:id/namespaces", getK8sNamespaces)
		d.GET("/:id/pods", getK8sPods)
		d.POST("/test-k8s", testK8sConnection)
		d.GET("/models", listLLMModels)
	}
}

func listIntegrations(ctx *gin.Context) {
	Service(ctx, func() (interface{}, error) {
		workspaceID := ctx.GetHeader("X-Workspace-Id")
		if workspaceID == "" {
			return nil, &APIError{Code: http.StatusBadRequest, Message: "X-Workspace-Id header is required"}
		}

		integrationType := ctx.Query("type")
		return services.Integration.List(ctx, workspaceID, integrationType)
	})
}

func getIntegration(ctx *gin.Context) {
	Service(ctx, func() (interface{}, error) {
		id := ctx.Param("id")
		if id == "" {
			return nil, &APIError{Code: http.StatusBadRequest, Message: "integration id is required"}
		}
		return services.Integration.Get(ctx, id)
	})
}

func updateIntegration(ctx *gin.Context) {
	r := new(types.RequestUpdateIntegration)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}
	r.ID = ctx.Param("id")
	if r.ID == "" {
		Error(ctx, nil, "integration id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Integration.Update(ctx, r)
	})
}

func getK8sNamespaces(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "integration id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Integration.GetNamespaces(ctx, id)
	})
}

func getK8sPods(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "integration id is required")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		Error(ctx, nil, "namespace query parameter is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Integration.GetPods(ctx, id, namespace)
	})
}

func testK8sConnection(ctx *gin.Context) {
	r := new(types.RequestTestK8sConnection)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}
	if r.Kubeconfig == "" {
		Error(ctx, nil, "kubeconfig is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Integration.TestK8sConnection(ctx, r.Kubeconfig)
	})
}

func listLLMModels(ctx *gin.Context) {
	Service(ctx, func() (interface{}, error) {
		workspaceID := ctx.GetHeader("X-Workspace-Id")
		if workspaceID == "" {
			return nil, &APIError{Code: http.StatusBadRequest, Message: "X-Workspace-Id header is required"}
		}
		return services.Integration.ListModels(ctx, workspaceID)
	})
}
