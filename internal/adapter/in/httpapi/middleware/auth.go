package middleware

import (
	"context"
	"crypto/rsa"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type administratorIDKey struct{}

// RequireAuth accepts exactly two algorithms, each with its own dedicated key
// — never "whatever alg the token declares" against one shared secret, which
// is the classic hole (rules/2-anexos/C-api-hexagonal.md, numeral 5.3.7,
// "Errores frecuentes"): RS256 (a real Administrator session, verified with
// lms-access-api's public key) or HS256 (an internal service-to-service call,
// verified with a secret this service never uses for anything else). Because
// the keyfunc below branches on token.Method before it ever returns key
// material, an attacker can't take the public key's own bytes and replay them
// as an HMAC secret to forge a signature — that confusion only works if a
// single keyfunc returns the same value regardless of algorithm.
func RequireAuth(publicKey *rsa.PublicKey, internalSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeUnauthorized(w)
				return
			}

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
				switch t.Method.(type) {
				case *jwt.SigningMethodRSA:
					return publicKey, nil
				case *jwt.SigningMethodHMAC:
					return []byte(internalSecret), nil
				default:
					return nil, jwt.ErrTokenSignatureInvalid
				}
			}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg(), jwt.SigningMethodHS256.Alg()}))
			if err != nil || !token.Valid {
				writeUnauthorized(w)
				return
			}

			administratorID, _ := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), administratorIDKey{}, administratorID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdministratorID retrieves the authenticated Administrator's ID from context.
func AdministratorID(ctx context.Context) string {
	id, _ := ctx.Value(administratorIDKey{}).(string)
	return id
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Authentication token required"}`))
}
