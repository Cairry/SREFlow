package api

import (
	"net/http"
	"RCAgent/internal/services"
	"RCAgent/internal/types"

	"github.com/gin-gonic/gin"
)

type datasource struct{}

var DatasourceController = new(datasource)

func (datasource) API(gin *gin.RouterGroup) {
	d := gin.Group("datasources")
	{
		d.GET("", listDatasources)
		d.POST("", createDatasource)
		d.GET("/:id", getDatasource)
		d.PUT("/:id", updateDatasource)
		d.DELETE("/:id", deleteDatasource)
		d.GET("/:id/namespaces", getK8sNamespaces)
		d.GET("/:id/pods", getK8sPods)
		d.POST("/test-k8s", testK8sConnection)
	}
}

func listDatasources(ctx *gin.Context) {
	Service(ctx, func() (interface{}, error) {
		workspaceID := ctx.GetHeader("X-Workspace-Id")
		if workspaceID == "" {
			return nil, &APIError{Code: http.StatusBadRequest, Message: "X-Workspace-Id header is required"}
		}

		datasourceType := ctx.Query("type")
		return services.Datasource.List(ctx, workspaceID, datasourceType)
	})
}

func getDatasource(ctx *gin.Context) {
	Service(ctx, func() (interface{}, error) {
		id := ctx.Param("id")
		if id == "" {
			return nil, &APIError{Code: http.StatusBadRequest, Message: "datasource id is required"}
		}
		return services.Datasource.Get(ctx, id)
	})
}

func createDatasource(ctx *gin.Context) {
	r := new(types.RequestCreateDatasource)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}
	if err := r.Valid(nil); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	workspaceID := ctx.GetHeader("X-Workspace-Id")
	if workspaceID == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Datasource.Create(ctx, r, workspaceID)
	})
}

func updateDatasource(ctx *gin.Context) {
	r := new(types.RequestUpdateDatasource)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}
	r.ID = ctx.Param("id")
	if r.ID == "" {
		Error(ctx, nil, "datasource id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Datasource.Update(ctx, r)
	})
}

func deleteDatasource(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "datasource id is required")
		return
	}
	workspaceID := ctx.GetHeader("X-Workspace-Id")
	if workspaceID == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Datasource.Delete(ctx, id, workspaceID)
	})
}

func getK8sNamespaces(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "datasource id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Datasource.GetNamespaces(ctx, id)
	})
}

func getK8sPods(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "datasource id is required")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		Error(ctx, nil, "namespace query parameter is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Datasource.GetPods(ctx, id, namespace)
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
		return services.Datasource.TestK8sConnection(ctx, r.Kubeconfig)
	})
}
