package debug

import (
	"context"

	"github.com/MetaEMK/FGK_PASMAS_backend/database"
	internalerror "github.com/MetaEMK/FGK_PASMAS_backend/internalError"
	"github.com/MetaEMK/FGK_PASMAS_backend/logging"
)

// TODO: REMOVE THIS THING: THIS IS FOR DEBUG PURPOSES ONLY
// IMPORTANT: DO NOT USE THIS IN PRODUCTION

var log = logging.DbDebugLogger

type intErr = internalerror.InternalError
var mode = "DEBUG"

// TruncateDatabase truncates the database and seeds it with default values
func TruncateDatabase() error {
    if mode!= "DEBUG" {
        log.Debug(mode)
        return intErr{Type: internalerror.ErrorUnknownError, Message: "This is functionality is only allowed in DEBUG mode"}
    }

    log.Warn("TRUNCATING DATABASE")

    connErr := database.CheckDatabaseConnection()
    if connErr != nil {
        return intErr{Type: internalerror.ErrorDatabaseConnectionError, Message: "Failed to connect to database"}
    }

    query := `
        truncate table passenger restart identity cascade;
        truncate table division restart identity cascade; `

    _, err := database.PgConn.Exec(context.Background(), query)

    if err != nil {
        return intErr{Type: internalerror.ErrorDatabaseQueryError, Message: "Could not run TRUNCATE Statements", Body: err}
    }

    log.Warn("TRUNCATING FINISHED - seeding")
    database.SeedDatabase()

    return nil
}
