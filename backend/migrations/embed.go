package migrations

import "embed"

// FS contains the SQL migrations shipped with the backend binary.
//
// Keeping the files in the binary makes startup independent of the process
// working directory and ensures the deployed code and schema steps match.
//
//go:embed *.sql
var FS embed.FS
