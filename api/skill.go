package api

import (
	"SREFlow/internal/services"
	"SREFlow/internal/types"

	"github.com/gin-gonic/gin"
)

type skillServer struct{}

var SkillController = new(skillServer)

func (skillServer) API(gin *gin.RouterGroup) {
	g := gin.Group("skills")
	{
		g.GET("", listSkills)
		g.POST("", createSkill)
		g.GET("/:id", getSkill)
		g.PUT("/:id", updateSkill)
		g.DELETE("/:id", deleteSkill)
	}
}

func listSkills(ctx *gin.Context) {
	workspaceID := ctx.GetHeader("X-Workspace-Id")
	if workspaceID == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Skill.List(ctx, workspaceID)
	})
}

func getSkill(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Skill.Get(ctx, id)
	})
}

func createSkill(ctx *gin.Context) {
	workspaceID := ctx.GetHeader("X-Workspace-Id")
	if workspaceID == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	r := new(types.RequestCreateSkill)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	if err := r.Valid(); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Skill.Create(ctx, r, workspaceID)
	})
}

func updateSkill(ctx *gin.Context) {
	r := new(types.RequestUpdateSkill)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	r.ID = ctx.Param("id")
	if r.ID == "" {
		Error(ctx, nil, "id is required")
		return
	}

	if err := r.Valid(); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Skill.Update(ctx, r)
	})
}

func deleteSkill(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Skill.Delete(ctx, id)
	})
}
