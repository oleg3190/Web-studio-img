package auth

import (
    "context"
    "errors"
    "time"
    "github.com/google/uuid"
)
var ErrSessionRevoked = errors.New("session revoked")
type User struct { ID uuid.UUID; Email, Name, PasswordHash string }
type Store interface {
    FindUserByEmail(context.Context,string)(User,error)
    CreateUser(context.Context,string,string,string)(User,error)
    CreateSession(context.Context,uuid.UUID,string,time.Time)(uuid.UUID,error)
    IsSessionActive(context.Context,uuid.UUID,time.Time)(bool,error)
    RevokeSession(context.Context,uuid.UUID,time.Time) error
}
