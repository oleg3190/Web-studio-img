package auth

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "strings"
    "time"
    "github.com/google/uuid"
)
type Service struct { store Store; tokens *TokenManager; now func() time.Time }
func NewService(store Store,tokens *TokenManager)(*Service,error){
    if store==nil || tokens==nil{return nil,errors.New("auth service requires store and token manager")}
    return &Service{store:store,tokens:tokens,now:time.Now},nil
}
func(s *Service) Register(ctx context.Context,email,name,password string)(User,string,time.Time,error){
    email=strings.ToLower(strings.TrimSpace(email)); name=strings.TrimSpace(name)
    if email==""||name==""{return User{},"",time.Time{},errors.New("email and name are required")}
    hash,err:=HashPassword(password);if err!=nil{return User{},"",time.Time{},err}
    user,err:=s.store.CreateUser(ctx,email,name,hash);if err!=nil{return User{},"",time.Time{},err}
    return s.issueSession(ctx,user)
}
func(s *Service) Login(ctx context.Context,email,password string)(User,string,time.Time,error){
    user,err:=s.store.FindUserByEmail(ctx,strings.ToLower(strings.TrimSpace(email)))
    if err!=nil||CheckPassword(user.PasswordHash,password)!=nil{return User{},"",time.Time{},ErrInvalidCredentials}
    return s.issueSession(ctx,user)
}
func(s *Service) Logout(ctx context.Context,sessionID uuid.UUID)error{return s.store.RevokeSession(ctx,sessionID,s.now())}
func(s *Service) issueSession(ctx context.Context,user User)(User,string,time.Time,error){
    sid:=uuid.New();now:=s.now();token,expires,err:=s.tokens.Issue(user.ID,sid,now)
    if err!=nil{return User{},"",time.Time{},err};h:=sha256.Sum256([]byte(token))
    if _,err=s.store.CreateSession(ctx,user.ID,hex.EncodeToString(h[:]),expires);err!=nil{return User{},"",time.Time{},err}
    return user,token,expires,nil
}
func TokenHash(token string)string{h:=sha256.Sum256([]byte(token));return hex.EncodeToString(h[:])}
