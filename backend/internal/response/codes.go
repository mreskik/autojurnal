package response

// Kode response. 0 = sukses, error dimulai dari 1000.
//
//	1000-1099 : umum
//	1100-1199 : master users
//	1200-1299 : auth
//	1300-1399 : runners
//	1400-1499 : journals
//	1500-1599 : trades
const (
	CodeOK = 0

	// umum
	CodeInternalError       = 1000
	CodeInvalidRequestBody  = 1001
	CodeValidationFailed    = 1002
	CodeInvalidParam        = 1003
	CodeNotFound            = 1004
	CodeDatabaseUnavailable = 1005
	CodeTooManyRequests     = 1006

	// master users
	CodeUserNotFound  = 1100
	CodeUsernameTaken = 1101
	CodeEmailTaken    = 1102

	// auth
	CodeInvalidCredentials = 1200
	CodeUnauthorized       = 1201
	CodeSessionNotFound    = 1202

	// runners
	CodeRunnerNotFound     = 1300
	CodeInvalidRunnerToken = 1301
	CodeRunnerTokenExpired = 1302
	CodeRunnerInactive     = 1303

	// journals
	CodeJournalNotFound     = 1400
	CodeTransactionNotFound = 1401
	CodeRunnerAlreadyJoined = 1402
	CodeRunnerNotInJournal  = 1403
	CodeInsufficientBalance = 1404
	CodeJournalLocked       = 1405
	CodeJournalArchived     = 1406
	CodeInvalidJournalToken = 1407
	CodeRunnerNotJoined     = 1408

	// trades
	CodeTradeNotFound      = 1500
	CodeTradeAlreadyClosed = 1501
	CodeTradeNotOwned      = 1502
	CodeOpenTradeExists    = 1503
)
