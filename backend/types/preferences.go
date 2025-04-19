package types

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/downinfo"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/pkg/proxy"
)

type Preferences struct {
	Behavior     PreferencesBehavior `json:"behavior" yaml:"behavior"`
	General      PreferencesGeneral  `json:"general" yaml:"general"`
	Proxy        proxy.Config        `json:"proxy" yaml:"proxy"`
	Download     downinfo.Config     `json:"download" yaml:"download"`
	Logger       logger.Config       `json:"logger" yaml:"logger"`
	Dependencies Dependencies        `json:"dependencies" yaml:"dependencies"`
	ListendInfo  ListendInfo         `json:"listendInfo" yaml:"listend_info"`
}

func NewPreferences() Preferences {
	return Preferences{
		Behavior: PreferencesBehavior{
			WindowWidth:  consts.DEFAULT_WINDOW_WIDTH,
			WindowHeight: consts.DEFAULT_WINDOW_HEIGHT,
		},
		General: PreferencesGeneral{
			Theme:       "auto",
			Language:    "auto",
			CheckUpdate: true,
		},
		Proxy: proxy.Config{
			Type: "system", // default use system proxy
		},
		Download: downinfo.Config{
			Dir: downinfo.GetDefaultDownloadDir(),
		},
		Logger: *logger.DefaultConfig(),
		Dependencies: Dependencies{
			YTDLP: SoftwareInfo{
				Version: consts.YTDLP_VERSION,
			},
			FFMpeg: SoftwareInfo{},
		},
		ListendInfo: ListendInfo{
			WS: ListendInfoDetails{
				Protocol: "ws",
				IP:       "localhost",
				Port:     consts.WS_PORT,
				Path:     "ws",
			},
			MCP: ListendInfoDetails{
				Protocol: "http",
				IP:       "localhost",
				Port:     consts.MCP_SERVER_PORT,
				Path:     "sse",
			},
		},
	}
}

type PreferencesBehavior struct {
	Welcomed        bool `json:"welcomed" yaml:"welcomed"`
	WindowWidth     int  `json:"windowWidth" yaml:"window_width"`
	WindowHeight    int  `json:"windowHeight" yaml:"window_height"`
	WindowMaximised bool `json:"windowMaximised" yaml:"window_maximised"`
	WindowPosX      int  `json:"windowPosX" yaml:"window_pos_x"`
	WindowPosY      int  `json:"windowPosY" yaml:"window_pos_y"`
}

type PreferencesGeneral struct {
	Theme       string `json:"theme" yaml:"theme"`
	Language    string `json:"language" yaml:"language"`
	CheckUpdate bool   `json:"checkUpdate" yaml:"check_update"`
	SkipVersion string `json:"skipVersion" yaml:"skip_version,omitempty"`
}

type Dependencies struct {
	YTDLP  SoftwareInfo `json:"ytdlp" yaml:"ytdlp"`
	FFMpeg SoftwareInfo `json:"ffmpeg" yaml:"ffmpeg"`
}

type ListendInfo struct {
	WS  ListendInfoDetails `json:"ws" yaml:"ws"`
	MCP ListendInfoDetails `json:"mcp" yaml:"mcp"`
}

type ListendInfoDetails struct {
	Protocol string `json:"protocol" yaml:"protocol"`
	IP       string `json:"ip" yaml:"ip"`
	Port     int    `json:"port" yaml:"port"`
	Path     string `json:"path" yaml:"path"`
}
