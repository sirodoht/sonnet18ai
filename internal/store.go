package internal

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = fmt.Errorf("not found")

type SQLStore struct {
	db *gorm.DB
}

func NewSQLStore(gdb *gorm.DB) *SQLStore {
	err := gdb.AutoMigrate(
		&Document{},
	)
	if err != nil {
		panic(err)
	}

	return &SQLStore{
		db: gdb,
	}
}

func (s *SQLStore) PutDocument(
	ctx context.Context,
	req *Document,
) error {
	if req == nil {
		return fmt.Errorf("failed to put document: nil request")
	}

	err := s.db.
		WithContext(ctx).
		Clauses(
			clause.OnConflict{
				UpdateAll: true,
			},
		).
		Create(req).
		Error
	if err != nil {
		return fmt.Errorf("failed to put document: %w", err)
	}

	return nil
}

func (s *SQLStore) GetDocuments(
	ctx context.Context,
) ([]*Document, error) {
	query := s.db.WithContext(ctx)

	var documents []*Document
	err := query.
		Order("created_at DESC").
		Find(&documents).
		Error
	if err != nil {
		return nil, fmt.Errorf("failed to get document list: %w", err)
	}

	return documents, nil
}

func (s *SQLStore) GetDocument(
	ctx context.Context,
	id uint,
) (*Document, error) {
	query := s.db.WithContext(ctx)

	var document Document
	err := query.
		Where("document_id = ?", id).
		Find(&document).
		Error
	if err != nil {
		return nil, fmt.Errorf("failed to get document single: %w", err)
	}

	return &document, nil
}
