package auth

import (
	"context"
	"net/http"

	"github.com/sm-operator/sapcp-operator/internal/httputil"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// HTTPClient interface
//go:generate counterfeiter . HTTPClient
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewAuthClient(ccConfig *clientcredentials.Config, sslDisabled bool) HTTPClient {
	httpClient := httputil.BuildHTTPClient(sslDisabled)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	return oauth2.NewClient(ctx, ccConfig.TokenSource(ctx))
}
