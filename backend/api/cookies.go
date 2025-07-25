package api

import (
	"CanMe/backend/core/downtasks"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/types"
	"context"
	"encoding/json"

	"go.uber.org/zap"
)

// CookiesAPI handles browser cookie related operations.
type CookiesAPI struct {
	ctx              context.Context
	downtasksService *downtasks.Service
}

// NewCookiesAPI creates a new CookiesAPI instance.
func NewCookiesAPI(downtasksService *downtasks.Service) *CookiesAPI {
	return &CookiesAPI{
		downtasksService: downtasksService,
	}
}

// WailsInit is called at application startup.
func (a *CookiesAPI) WailsInit(ctx context.Context) error {
	a.ctx = ctx
	return nil
}

// RefreshCookies triggers a refresh of the cookie cache.
func (a *CookiesAPI) SyncCookies(syncFrom string, browsers []string) types.JSResp {
	logger.GetLogger().Debug("Starting browser cookies sync...")

	// 立即启动异步同步，不等待结果
	a.downtasksService.SyncCookies(syncFrom, browsers)

	// 立即返回成功响应
	logger.GetLogger().Debug("Browser cookies sync started successfully.")
	return types.JSResp{
		Success: true,
		Msg:     "Cookie sync started, you will be notified when completed",
		Data:    nil,
	}
}

func (a *CookiesAPI) GetBrowserByDomain(targetURL string) types.JSResp {
	logger.GetLogger().Debug("Getting browser for URL", zap.String("url", targetURL))
	browser, err := a.downtasksService.GetBrowserByDomain(targetURL)
	if err != nil {
		logger.GetLogger().Error("Failed to get browser for URL", zap.String("url", targetURL), zap.Error(err))
		return types.JSResp{
			Success: false,
			Msg:     "Failed to get browser: " + err.Error(),
		}
	}

	data, _ := json.Marshal(browser)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

// GetCookiesByDomain retrieves cookies for a specific URL from a given browser.
func (a *CookiesAPI) GetCookiesByDomain(browser string, targetURL string) types.JSResp {
	logger.GetLogger().Debug("Getting cookies for URL", zap.String("url", targetURL), zap.String("browser", browser))
	cookies, err := a.downtasksService.GetCookiesByDomain(browser, targetURL)
	if err != nil {
		logger.GetLogger().Error("Failed to get cookies for URL", zap.String("url", targetURL), zap.String("browser", browser), zap.Error(err))
		return types.JSResp{
			Success: false,
			Msg:     "Failed to get cookies: " + err.Error(),
		}
	}

	data, _ := json.Marshal(cookies)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}

// ListAllCookies retrieves all cached cookies, grouped by browser.
func (a *CookiesAPI) ListAllCookies() types.JSResp {
	logger.GetLogger().Debug("Listing all cached cookies by browser.")
	cookies, err := a.downtasksService.ListAllCookies()
	if err != nil {
		logger.GetLogger().Error("Failed to list all cookies: " + err.Error())
		return types.JSResp{
			Success: false,
			Msg:     "Failed to list all cookies: " + err.Error(),
		}
	}

	data, _ := json.Marshal(cookies)
	return types.JSResp{
		Success: true,
		Data:    string(data),
	}
}
