package internal

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

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

func (s *SQLStore) CreateDocument(
	ctx context.Context,
) (uint, error) {
	query := s.db.WithContext(ctx)

	document := &Document{
		Title: "Untitled",
	}

	// save new document
	err := query.Save(document).Error
	if err != nil {
		return 0, fmt.Errorf("failed to save document: %w", err)
	}

	return document.DocumentID, nil
}

func (s *SQLStore) DeleteDocument(
	ctx context.Context,
	id uint,
) error {
	query := s.db.WithContext(ctx)

	err := query.Delete(&Document{
		DocumentID: id,
	}).Error
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

func (s *SQLStore) UpdateDocument(
	ctx context.Context,
	id uint,
	newTitle string,
	newBody string,
) error {
	query := s.db.WithContext(ctx)

	// retrieve the document to update
	var document Document
	err := query.
		Where("document_id = ?", id).
		Find(&document).
		Error
	if err != nil {
		return fmt.Errorf("failed to find document to update: %w", err)
	}

	// update document values
	document.Title = newTitle
	document.Body = newBody

	// save updated document
	err = query.Save(&document).Error
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
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
