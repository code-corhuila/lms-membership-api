// Package circulation implements the membership.ActiveLoansChecker driven
// port over HTTP — the replacement for the in-process LoanRepository call
// that DeactivateStudent used before the microservices split.
package circulation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Client calls circulation-service's own /api/v1/loans endpoint (the same one
// HU-07's UI uses) filtered to a single student's active loans, and reads the
// total from the paginated envelope. There is no dedicated internal endpoint —
// reusing the public one keeps circulation-service from having to maintain a
// second, internal-only API surface for a single count.
type Client struct {
	baseURL   string
	jwtSecret string
	http      *http.Client
}

func NewClient(baseURL, jwtSecret string) *Client {
	return &Client{
		baseURL:   baseURL,
		jwtSecret: jwtSecret,
		http:      &http.Client{Timeout: 5 * time.Second},
	}
}

type loansResponse struct {
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
}

func (c *Client) CountActive(ctx context.Context, studentID string) (int, error) {
	token, err := c.mintServiceToken()
	if err != nil {
		return 0, fmt.Errorf("minting internal service token: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/loans?studentId=%s&status=ACTIVE", c.baseURL, url.QueryEscape(studentID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("calling circulation-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("circulation-service returned %d", resp.StatusCode)
	}

	var body loansResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decoding circulation-service response: %w", err)
	}
	return body.Meta.Total, nil
}

// mintServiceToken signs a short-lived internal token with the JWT_SECRET
// shared by every service — the same secret access-service uses to issue
// Administrator tokens. Circulation-service's RequireAuth middleware can't
// tell this apart from a real Administrator session (v1 has no service-to-
// service auth scope), which is an accepted trade-off for this project's size.
func (c *Client) mintServiceToken() (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub": "membership-service",
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.jwtSecret))
}
