package service

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	TokenType string `json:"token_type"` 
	jwt.RegisteredClaims
}

func NewJWTService(privateKeyB64, publicKeyB64 string) (*JWTService, error) {
	privPEM, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	pubPEM, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}

	privBlock, _ := pem.Decode(privPEM)
	if privBlock == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		privInterface, err := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		var ok bool
		privKey, ok = privInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
	}
	pubBlock, _ := pem.Decode(pubPEM)
	if pubBlock == nil {
		return nil, fmt.Errorf("failed to decode public key PEM")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return &JWTService{privateKey: privKey, publicKey: pubKey}, nil
}

func (s *JWTService) GenerateAccessToken(userID uuid.UUID, email string, duration time.Duration) (string, error) {
    claims := Claims{
        UserID:    userID.String(),
        Email:     email,
        TokenType: "access",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.privateKey)
}


func (s *JWTService) GenerateRefreshToken(userID uuid.UUID, duration time.Duration) (string, error) {
    claims := Claims{
        UserID:    userID.String(),
        TokenType: "refresh",
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   userID.String(),
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.privateKey)
}

func (s *JWTService) ValidateAccessToken(tokenStr string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return s.publicKey, nil
    })
    if err != nil {
        return nil, err
    }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }
    if claims.TokenType != "access" {
        return nil, fmt.Errorf("invalid token type")
    }
    return claims, nil
}

func (s *JWTService) ValidateRefreshToken(tokenStr string) (uuid.UUID, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        return s.publicKey, nil
    })
    if err != nil {
        return uuid.Nil, err
    }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return uuid.Nil, fmt.Errorf("invalid token")
    }
    if claims.TokenType != "refresh" {
        return uuid.Nil, fmt.Errorf("invalid token type")
    }
    return uuid.Parse(claims.UserID)
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
