package ctx

import (
	"SREFlow/internal/cache"
	"SREFlow/internal/repos"
	"context"

	"github.com/zeromicro/go-zero/core/logc"
)

type Context struct {
	Ctx      context.Context
	Database repos.EntryInter
	Cache    cache.EntryInter
}

var (
	Ctx      context.Context
	Database repos.EntryInter
	Cache    cache.EntryInter
)

func NewContext(ctx context.Context, database repos.EntryInter, cache cache.EntryInter) *Context {
	Ctx = ctx
	Database = database
	Cache = cache
	logc.Info(context.Background(), "全局 Context 初始化完成!")
	return &Context{
		Ctx:      ctx,
		Database: database,
		Cache:    cache,
	}
}

func DO() *Context {
	return &Context{
		Ctx:      Ctx,
		Database: Database,
		Cache:    Cache,
	}
}
