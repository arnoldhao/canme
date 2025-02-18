package preferences

import (
	"context"
	"net"
	"net/http"
	"time"

	"CanMe/backend/consts"
	"CanMe/backend/pkg/specials/proxy"
	"CanMe/backend/types"
)

// SetProxy set proxy
func (s *Service) SetProxy(proxyAddr string) (resp types.JSResp) {
	err := proxy.GetInstance().SetProxy(proxyAddr)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
	return
}

// GetProxy get proxy
func (s *Service) GetProxy() string {
	if proxy := proxy.GetInstance().GetProxyURL(); proxy != nil {
		return proxy.String()
	}
	return ""
}

func (s *Service) TestProxy(id string) (err error) {
	// define sites to test
	testSites := []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}{
		{"Google", "https://www.google.com"},
		{"YouTube", "https://www.youtube.com"},
		{"Bilibili", "https://www.bilibili.com"},
		{"ChatGPT", "https://www.chatgpt.com"},
		{"GitHub", "https://github.com"},
	}

	// create http client
	proxyInstance := proxy.GetInstance()
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	if proxyURL := proxyInstance.GetProxyURL(); proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// test each site
	go func() {
		for i, site := range testSites {
			done := i == len(testSites)-1
			ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
			startTime := time.Now()

			req, err := http.NewRequestWithContext(ctx, "GET", site.URL, nil)
			if err != nil {
				s.wsService.SendToClient(types.WSResponse{
					Namespace: consts.NAMESPACE_PROXY,
					Event:     consts.EVENT_PROXY_TEST_RESULT_ERROR,
					Data: types.TestProxyResult{
						ID:      id,
						Done:    done,
						URL:     site.URL,
						Success: false,
						Error:   err.Error(),
					},
				})
				cancel()
				continue
			}

			response, err := client.Do(req)
			if err != nil {
				s.wsService.SendToClient(types.WSResponse{
					Namespace: consts.NAMESPACE_PROXY,
					Event:     consts.EVENT_PROXY_TEST_RESULT_ERROR,
					Data: types.TestProxyResult{
						ID:      id,
						Done:    done,
						URL:     site.URL,
						Success: false,
						Error:   "connection failed",
					},
				})
				cancel()
				continue
			}

			latency := time.Since(startTime).Milliseconds()
			s.wsService.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_PROXY,
				Event:     consts.EVENT_PROXY_TEST_RESULT,
				Data: types.TestProxyResult{
					ID:      id,
					Done:    done,
					URL:     site.URL,
					Success: response.StatusCode == http.StatusOK,
					Latency: int(latency),
					Error:   "",
				},
			})

			response.Body.Close()
			cancel()
		}
		// send complated
		s.wsService.SendToClient(types.WSResponse{
			Namespace: consts.NAMESPACE_PROXY,
			Event:     consts.EVENT_PROXY_TEST_RESULT_COMPLETED,
			Data: types.TestProxyResult{
				ID:    id,
				Error: "",
			},
		})
	}()

	return
}
