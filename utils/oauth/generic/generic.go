package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/komari-monitor/komari/utils"
	"github.com/komari-monitor/komari/utils/oauth/factory"
	"github.com/patrickmn/go-cache"
)

func (g *Generic) GetName() string {
	return "generic"
}
func (g *Generic) GetConfiguration() factory.Configuration {
	return &g.Addition
}

func (g *Generic) GetAuthorizationURL() (string, string) {
	state := utils.GenerateRandomString(16)

	// 构建GitHub OAuth授权URL
	authURL := fmt.Sprintf(
		"%s?client_id=%s&state=%s&scope=%s&response_type=code",
		g.Addition.AuthURL,
		url.QueryEscape(g.Addition.ClientId),
		url.QueryEscape(state),
		url.QueryEscape(g.Addition.Scope),
	)
	g.stateCache.Set(state, true, cache.DefaultExpiration)
	return authURL, state
}
func (g *Generic) OnCallback(ctx context.Context, state string, query map[string]string) (factory.OidcCallback, error) {
	code := query["code"]

	// 验证state防止CSRF攻击
	if g.stateCache == nil {
		return factory.OidcCallback{}, fmt.Errorf("state cache not initialized")
	}
	if _, ok := g.stateCache.Get(state); !ok {
		return factory.OidcCallback{}, fmt.Errorf("invalid state")
	}
	if state == "" {
		return factory.OidcCallback{}, fmt.Errorf("invalid state")
	}

	// 获取code
	if code == "" {
		return factory.OidcCallback{}, fmt.Errorf("no code provided")
	}

	// 获取访问令牌
	data := url.Values{
		"client_id":     {g.Addition.ClientId},
		"client_secret": {g.Addition.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}

	req, _ := http.NewRequest("POST", g.Addition.TokenURL, nil)
	req.URL.RawQuery = data.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return factory.OidcCallback{}, fmt.Errorf("failed to get access token: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return factory.OidcCallback{}, fmt.Errorf("failed to parse access token response: %v", err)
	}

	// 获取用户信息
	userReq, _ := http.NewRequest("GET", g.Addition.UserInfoURL, nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	userReq.Header.Set("Accept", "application/json")

	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		return factory.OidcCallback{}, fmt.Errorf("failed to get user info: %v", err)
	}
	defer userResp.Body.Close()

	var user map[string]interface{}
	if err := json.NewDecoder(userResp.Body).Decode(&user); err != nil {
		return factory.OidcCallback{}, fmt.Errorf("failed to parse user info response: %v", err)
	}

	userId, ok := user[g.Addition.UserIDField]
	if !ok {
		return factory.OidcCallback{}, fmt.Errorf("user id field '%s' not found in user info response", g.Addition.UserIDField)
	}

	return factory.OidcCallback{UserId: fmt.Sprintf("%v", userId)}, nil
}
func (g *Generic) Init() error {
	g.stateCache = cache.New(time.Minute*5, time.Minute*10)
	return nil
}
func (g *Generic) Destroy() error {
	g.stateCache.Flush()
	return nil
}

var _ factory.IOidcProvider = (*Generic)(nil)
