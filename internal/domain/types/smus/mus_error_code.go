package smus

// MUS protocol error codes.
// Based on OpenSMUS MUSErrorCode.java (MIT License, Mauricio Piacentini).
// Values are sequential int32 starting at 0x800414A1.
const (
	ErrNoError                          int32 = 0x00000000
	ErrUnknown                          int32 = -0x7FFBEB5F // 0x800414A1
	ErrInvalidMovieID                   int32 = -0x7FFBEB5E // 0x800414A2
	ErrInvalidUserID                    int32 = -0x7FFBEB5D // 0x800414A3
	ErrInvalidPassword                  int32 = -0x7FFBEB5C // 0x800414A4
	ErrIncomingDataLost                 int32 = -0x7FFBEB5B // 0x800414A5
	ErrInvalidServerName                int32 = -0x7FFBEB5A // 0x800414A6
	ErrNoConnectionsAvailable           int32 = -0x7FFBEB59 // 0x800414A7
	ErrBadParameter                     int32 = -0x7FFBEB58 // 0x800414A8
	ErrNoSocketManager                  int32 = -0x7FFBEB57 // 0x800414A9
	ErrNoCurrentConnection              int32 = -0x7FFBEB56 // 0x800414AA
	ErrNoWaitingMessage                 int32 = -0x7FFBEB55 // 0x800414AB
	ErrBadConnectionID                  int32 = -0x7FFBEB54 // 0x800414AC
	ErrWrongNumberOfParams              int32 = -0x7FFBEB53 // 0x800414AD
	ErrUnknownInternalError             int32 = -0x7FFBEB52 // 0x800414AE
	ErrConnectionRefused                int32 = -0x7FFBEB51 // 0x800414AF
	ErrMessageTooLarge                  int32 = -0x7FFBEB50 // 0x800414B0
	ErrInvalidMessageFormat             int32 = -0x7FFBEB4F // 0x800414B1
	ErrInvalidMessageLength             int32 = -0x7FFBEB4E // 0x800414B2
	ErrMessageMissing                   int32 = -0x7FFBEB4D // 0x800414B3
	ErrServerInitializationFailed       int32 = -0x7FFBEB4C // 0x800414B4
	ErrServerSendFailed                 int32 = -0x7FFBEB4B // 0x800414B5
	ErrServerCloseFailed                int32 = -0x7FFBEB4A // 0x800414B6
	ErrConnectionDuplicate              int32 = -0x7FFBEB49 // 0x800414B7
	ErrInvalidNumberOfMessageRecipients int32 = -0x7FFBEB48 // 0x800414B8
	ErrInvalidMessageRecipient          int32 = -0x7FFBEB47 // 0x800414B9
	ErrInvalidMessage                   int32 = -0x7FFBEB46 // 0x800414BA
	ErrServerInternalError              int32 = -0x7FFBEB45 // 0x800414BB
	ErrErrorJoiningGroup                int32 = -0x7FFBEB44 // 0x800414BC
	ErrErrorLeavingGroup                int32 = -0x7FFBEB43 // 0x800414BD
	ErrInvalidGroupName                 int32 = -0x7FFBEB42 // 0x800414BE
	ErrInvalidServerCommand             int32 = -0x7FFBEB41 // 0x800414BF
	ErrNotPermittedWithUserLevel        int32 = -0x7FFBEB40 // 0x800414C0
	ErrDatabaseError                    int32 = -0x7FFBEB3F // 0x800414C1
	ErrInvalidServerInitFile            int32 = -0x7FFBEB3E // 0x800414C2
	ErrDatabaseWrite                    int32 = -0x7FFBEB3D // 0x800414C3
	ErrDatabaseRead                     int32 = -0x7FFBEB3C // 0x800414C4
	ErrDatabaseUserIDNotFound           int32 = -0x7FFBEB3B // 0x800414C5
	ErrDatabaseAddUser                  int32 = -0x7FFBEB3A // 0x800414C6
	ErrDatabaseLocked                   int32 = -0x7FFBEB39 // 0x800414C7
	ErrDatabaseDataRecordNotUnique      int32 = -0x7FFBEB38 // 0x800414C8
	ErrDatabaseNoCurrentRecord          int32 = -0x7FFBEB37 // 0x800414C9
	ErrDatabaseRecordNotExists          int32 = -0x7FFBEB36 // 0x800414CA
	ErrDatabaseMovedPastLimits          int32 = -0x7FFBEB35 // 0x800414CB
	ErrDatabaseDataNotFound             int32 = -0x7FFBEB34 // 0x800414CC
	ErrDatabaseNoCurrentTag             int32 = -0x7FFBEB33 // 0x800414CD
	ErrDatabaseNoCurrentDB              int32 = -0x7FFBEB32 // 0x800414CE
	ErrDatabaseNoConfigurationFile      int32 = -0x7FFBEB31 // 0x800414CF
	ErrDatabaseRecordNotLocked          int32 = -0x7FFBEB30 // 0x800414D0
	ErrOperationNotAllowed              int32 = -0x7FFBEB2F // 0x800414D1
	ErrRequestedDataNotFound            int32 = -0x7FFBEB2E // 0x800414D2
	ErrMessageContainsErrorInfo         int32 = -0x7FFBEB2D // 0x800414D3
	ErrDataConcurrencyError             int32 = -0x7FFBEB2C // 0x800414D4
	ErrUDPSocketError                   int32 = -0x7FFBEB2B // 0x800414D5
)
