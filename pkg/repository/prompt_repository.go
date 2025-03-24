package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dskvich/ai-bot/pkg/domain"
	"github.com/uptrace/bun"
)

type promptRepository struct {
	db *bun.DB
}

func NewPromptRepository(db *bun.DB) *promptRepository {
	return &promptRepository{db: db}
}

func (p *promptRepository) Save(ctx context.Context, prompt *domain.Prompt) error {
	_, err := p.db.NewInsert().
		Model(prompt).
		Returning("id").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("saving prompt: %w", err)
	}

	return nil
}

func (p *promptRepository) GetByID(ctx context.Context, id int64) (*domain.Prompt, error) {
	var prompt domain.Prompt

	err := p.db.NewSelect().
		Model(&prompt).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("fetching prompt by id %d: %w", id, err)
	}

	return &prompt, nil
}
