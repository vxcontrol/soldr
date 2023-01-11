package client

import (
	"context"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/models"
	"soldr/internal/app/api/storage/mem"
	"soldr/internal/secret"
	"soldr/internal/storage"
	"soldr/internal/storage/mysql"
)

type AgentServerClient struct {
	db      *gorm.DB
	dbConns *mem.DBConnectionStorage
	s3Conns *mem.S3ConnectionStorage
}

func NewAgentServerClient(
	db *gorm.DB,
	serviceDBConns *mem.DBConnectionStorage,
	serviceS3Conns *mem.S3ConnectionStorage,
) *AgentServerClient {
	return &AgentServerClient{
		db:      db,
		dbConns: serviceDBConns,
		s3Conns: serviceS3Conns,
	}
}

func (c *AgentServerClient) GetDB(ctx context.Context, hash string) (*gorm.DB, error) {
	db, err := c.dbConns.Get(hash)
	if err != nil && err == mem.ErrNotFound {
		var service models.Service
		if err = c.db.Take(&service, "hash = ?", hash).Error; err != nil {
			return nil, fmt.Errorf("could not get service by hash '%s': %w", hash, err)
		}

		dsn := fmt.Sprintf("%s:%s@%s/%s?parseTime=true",
			service.Info.DB.User,
			service.Info.DB.Pass,
			fmt.Sprintf("tcp(%s:%d)", service.Info.DB.Host, service.Info.DB.Port),
			service.Info.DB.Name,
		)
		db, err := mysql.New(&mysql.Config{DSN: secret.NewString(dsn)})
		if err != nil {
			return nil, fmt.Errorf("could not create DB instance: %w", err)
		}
		if err = db.RetryConnect(ctx, 3, 100*time.Millisecond); err != nil {
			return nil, fmt.Errorf("could not connect to database: %w", err)
		}

		dbWithORM, err := db.WithORM(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not create ORM: %w", err)
		}

		c.dbConns.Set(hash, dbWithORM)
		return dbWithORM, nil
	}
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (c *AgentServerClient) GetS3(hash string) (storage.IStorage, error) {
	s3, err := c.s3Conns.Get(hash)
	if err != nil && err == mem.ErrNotFound {
		var service models.Service
		if err = c.db.Take(&service, "hash = ?", hash).Error; err != nil {
			return nil, fmt.Errorf("could not get service by hash '%s': %w", hash, err)
		}

		conn, err := storage.NewS3(service.Info.S3.ToS3ConnParams())
		if err != nil {
			return nil, fmt.Errorf("could not create S3 client: %w", err)
		}

		c.s3Conns.Set(hash, conn)
		return conn, nil
	}
	if err != nil {
		return nil, err
	}
	return s3, nil
}
