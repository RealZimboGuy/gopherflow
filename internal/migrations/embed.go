package migrations

import "embed"

//go:embed *
var FS embed.FS

// this is because embeddings do not allow ../ so we use this as a method to get it in other sub folders
