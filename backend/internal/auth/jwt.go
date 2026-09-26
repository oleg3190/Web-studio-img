package auth

import (
    "errors"
    "fmt"
    "strings"
    "time"
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type Claims struct {
    UserID uuid.UUID
    SessionID uuid.UUID
    jwt.RegisteredClaims
}
type TokenManager struct { secret []byte; issuer string; ttl time.Duration }

func NewTokenManager(secret, issuer string, ttl time.Duration) (*TokenManager, error) {
    if len(secret) < 32 { return nil, errors.New("JWT secret must be at least 32 bytes") }
    if ttl <= 0 { return nil, errors.New("JWT TTL must be positive") }
    if strings.TrimSpace(issuer) == "" { return nil, errors.New("JWT issuer must not be empty") }
    return &TokenManager{secret: []byte(secret), issuer: issuer, ttl: ttl}, nil
}
func (m *TokenManager) Issue(userID, sessionID uuid.UUID, now time.Time) (string, time.Time, error) {
    expires := now.Add(m.ttl)
    claims := Claims{UserID:userID, SessionID:sessionID, RegisteredClaims:jwt.RegisteredClaims{
        Issuer:m.issuer, Subject:userID.String(), ID:sessionID.String(),
        IssuedAt:jwt.NewNumericDate(now), ExpiresAt:jwt.NewNumericDate(expires), NotBefore:jwt.NewNumericDate(now),
    }}
    signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
    if err != nil { return "", time.Time{}, fmt.Errorf("sign access token: %w", err) }
    return signed, expires, nil
}
func (m *TokenManager) Parse(tokenString string, now time.Time) (Claims, error) {
    var claims Claims
    token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any,error) {
        if token.Method != jwt.SigningMethodHS256 { return nil, errors.New("unexpected signing method") }
        return m.secret,nil
    }, jwt.WithIssuer(m.issuer), jwt.WithLeeway(time.Second), jwt.WithTimeFunc(func() time.Time{return now}))
    if err != nil || !token.Valid { return Claims{}, errors.New("invalid access token") }
    uid, err := uuid.Parse(claims.Subject); if err != nil || uid != claims.UserID { return Claims{}, errors.New("invalid access token subject") }
    sid, err := uuid.Parse(claims.ID); if err != nil || sid != claims.SessionID { return Claims{}, errors.New("invalid access token session") }
    return claims,nil
}
