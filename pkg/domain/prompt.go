package domain

type Prompt struct {
	ID         int    `bun:",pk,autoincrement"`
	Text       string `bun:"text"`
	ImageBytes []byte `bun:"-"`
	AudioBytes []byte `bun:"-"`
}
