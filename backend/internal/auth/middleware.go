package auth

import (
    "context"
    "net/http"
    "strings"
    "time"
    "github.com/google/uuid"
)
type contextKey struct{}
type Principal struct { UserID uuid.UUID; SessionID uuid.UUID }
func WithAuth(manager *TokenManager,store Store,next http.Handler)http.Handler{
    return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        header:=strings.TrimSpace(r.Header.Get("Authorization"))
        if !strings.HasPrefix(header,"Bearer "){http.Error(w,"unauthorized",http.StatusUnauthorized);return}
        claims,err:=manager.Parse(strings.TrimSpace(strings.TrimPrefix(header,"Bearer ")),time.Now())
        if err!=nil{http.Error(w,"unauthorized",http.StatusUnauthorized);return}
        active,err:=store.IsSessionActive(r.Context(),claims.SessionID,time.Now())
        if err!=nil||!active{http.Error(w,"unauthorized",http.StatusUnauthorized);return}
        ctx:=context.WithValue(r.Context(),contextKey{},Principal{UserID:claims.UserID,SessionID:claims.SessionID})
        next.ServeHTTP(w,r.WithContext(ctx))
    })
}
func PrincipalFromContext(ctx context.Context)(Principal,bool){p,ok:=ctx.Value(contextKey{}).(Principal);return p,ok}
