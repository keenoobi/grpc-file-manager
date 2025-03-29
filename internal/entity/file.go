package entity

import "time"

type File struct {
	Name      string
	Size      int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Path      string
}
