package repos

import (
	"SREFlow/internal/i18n"
	"context"
	"fmt"

	"gorm.io/gorm"
)

// GormClient GORM数据库客户端
type GormClient struct {
	db *gorm.DB
}

// InterGormClient GORM客户端接口
type InterGormClient interface {
	// 基础CRUD操作
	Create(ctx context.Context, model interface{}) error
	Update(ctx context.Context, model interface{}, where map[string]interface{}) error
	UpdateFields(ctx context.Context, model interface{}, fields map[string]interface{}, where map[string]interface{}) error
	Delete(ctx context.Context, model interface{}, where map[string]interface{}) error

	// 查询操作
	First(ctx context.Context, dest interface{}, where map[string]interface{}) error
	Find(ctx context.Context, dest interface{}, where map[string]interface{}) error
	Count(ctx context.Context, model interface{}, where map[string]interface{}) (int64, error)
	Query(ctx context.Context, dest interface{}, where map[string]interface{}, opts ...QueryOption) (*PaginateResult, error)

	// 事务操作
	Begin(ctx context.Context) (InterGormClient, error)
	Commit() error
	Rollback() error
	Transaction(ctx context.Context, fn func(tx InterGormClient) error) error
}

// NewGormClient 创建GORM客户端
func NewGormClient(db *gorm.DB) InterGormClient {
	return &GormClient{db: db}
}

// Create 创建记录
func (g *GormClient) Create(ctx context.Context, model interface{}) error {
	return g.db.WithContext(ctx).Create(model).Error
}

// Update 更新记录（更新整个模型）
func (g *GormClient) Update(ctx context.Context, model interface{}, where map[string]interface{}) error {
	db := g.db.WithContext(ctx).Model(model)
	db = g.Where(db, where)
	return db.Updates(model).Error
}

// UpdateFields 更新指定字段
func (g *GormClient) UpdateFields(ctx context.Context, model interface{}, fields map[string]interface{}, where map[string]interface{}) error {
	db := g.db.WithContext(ctx).Model(model)
	db = g.Where(db, where)
	return db.Updates(fields).Error
}

// Delete 删除记录
func (g *GormClient) Delete(ctx context.Context, model interface{}, where map[string]interface{}) error {
	db := g.db.WithContext(ctx).Model(model)
	db = g.Where(db, where)
	return db.Delete(model).Error
}

// First 查询单条记录
func (g *GormClient) First(ctx context.Context, dest interface{}, where map[string]interface{}) error {
	db := g.db.WithContext(ctx)
	db = g.Where(db, where)
	return db.First(dest).Error
}

// Find 查询多条记录
func (g *GormClient) Find(ctx context.Context, dest interface{}, where map[string]interface{}) error {
	db := g.db.WithContext(ctx)
	db = g.Where(db, where)
	return db.Find(dest).Error
}

// Count 统计记录数量
func (g *GormClient) Count(ctx context.Context, model interface{}, where map[string]interface{}) (int64, error) {
	var count int64
	db := g.db.WithContext(ctx).Model(model)
	db = g.Where(db, where)
	err := db.Count(&count).Error
	return count, err
}

// Begin 开始事务
func (g *GormClient) Begin(ctx context.Context) (InterGormClient, error) {
	tx := g.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf(i18n.T("error.transactionFailed", map[string]interface{}{"error": tx.Error.Error()}))
	}
	return &GormClient{db: tx}, nil
}

// Commit 提交事务
func (g *GormClient) Commit() error {
	if err := g.db.Commit().Error; err != nil {
		return fmt.Errorf(i18n.T("error.transactionFailed", map[string]interface{}{"error": err.Error()}))
	}
	return nil
}

// Rollback 回滚事务
func (g *GormClient) Rollback() error {
	if err := g.db.Rollback().Error; err != nil {
		return fmt.Errorf(i18n.T("error.transactionFailed", map[string]interface{}{"error": err.Error()}))
	}
	return nil
}

// Transaction 执行事务（自动提交/回滚）
func (g *GormClient) Transaction(ctx context.Context, fn func(tx InterGormClient) error) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txClient := &GormClient{db: tx}
		return fn(txClient)
	})
}

// Where 应用where条件
func (g *GormClient) Where(db *gorm.DB, where map[string]interface{}) *gorm.DB {
	for column, value := range where {
		db = db.Where(column, value)
	}
	return db
}

// QueryOption 查询选项
type QueryOption struct {
	Page     int
	PageSize int
	OrderBy  string
}

// Query 通用查询方法，支持可选分页
func (g *GormClient) Query(ctx context.Context, dest interface{}, where map[string]interface{}, opts ...QueryOption) (*PaginateResult, error) {
	db := g.db.WithContext(ctx)
	db = g.Where(db, where)

	// 合并选项
	var option QueryOption
	if len(opts) > 0 {
		option = opts[0]
		// 如果有多个选项，合并它们
		for i := 1; i < len(opts); i++ {
			if opts[i].Page > 0 {
				option.Page = opts[i].Page
			}
			if opts[i].PageSize > 0 {
				option.PageSize = opts[i].PageSize
			}
			if opts[i].OrderBy != "" {
				option.OrderBy = opts[i].OrderBy
			}
		}
	}

	// 应用排序
	if option.OrderBy != "" {
		db = db.Order(option.OrderBy)
	}

	// 如果没有分页参数，直接查询所有数据
	if option.Page <= 0 || option.PageSize <= 0 {
		if err := db.Find(dest).Error; err != nil {
			return nil, fmt.Errorf(i18n.T("error.queryFailed", map[string]interface{}{"error": err.Error()}))
		}
		return &PaginateResult{
			Data:       dest,
			Total:      -1, // 表示未统计总数
			Page:       0,
			PageSize:   0,
			TotalPages: 0,
		}, nil
	}

	// 分页查询
	var total int64
	// 先统计总数
	countDB := db.Session(&gorm.Session{})
	if err := countDB.Count(&total).Error; err != nil {
		return nil, fmt.Errorf(i18n.T("error.queryFailed", map[string]interface{}{"error": err.Error()}))
	}

	// 计算偏移量
	offset := (option.Page - 1) * option.PageSize

	// 执行分页查询
	if err := db.Offset(offset).Limit(option.PageSize).Find(dest).Error; err != nil {
		return nil, fmt.Errorf(i18n.T("error.queryFailed", map[string]interface{}{"error": err.Error()}))
	}

	// 计算总页数
	totalPages := int(total) / option.PageSize
	if int(total)%option.PageSize > 0 {
		totalPages++
	}

	return &PaginateResult{
		Data:       dest,
		Total:      total,
		Page:       option.Page,
		PageSize:   option.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Paginate 分页查询结果
type PaginateResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
}
