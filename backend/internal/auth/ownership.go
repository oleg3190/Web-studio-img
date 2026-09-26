package auth

import (
    "context"
    "errors"
    "github.com/google/uuid"
)
var ErrForbidden=errors.New("forbidden")
type OwnershipStore interface{ProjectOwner(context.Context,uuid.UUID)(uuid.UUID,error)}
func RequireProjectOwner(ctx context.Context,store OwnershipStore,projectID uuid.UUID)error{
    principal,ok:=PrincipalFromContext(ctx);if !ok{return ErrForbidden}
    owner,err:=store.ProjectOwner(ctx,projectID);if err!=nil{return err}
    if owner!=principal.UserID{return ErrForbidden};return nil
}
