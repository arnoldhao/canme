package types

import "CanMe/backend/consts"

type Preferences struct {
	Behavior PreferencesBehavior `json:"behavior" yaml:"behavior"`
	General  PreferencesGeneral  `json:"general" yaml:"general"`
	Proxy    Proxy               `json:"proxy" yaml:"proxy"`
	Download Download            `json:"download" yaml:"download"`
}

func NewPreferences() Preferences {
	return Preferences{
		Behavior: PreferencesBehavior{
			AsideWidth:   consts.DEFAULT_ASIDE_WIDTH,
			WindowWidth:  consts.DEFAULT_WINDOW_WIDTH,
			WindowHeight: consts.DEFAULT_WINDOW_HEIGHT,
		},
		General: PreferencesGeneral{
			Theme:        "auto",
			Language:     "auto",
			FontSize:     consts.DEFAULT_FONT_SIZE,
			ScanSize:     consts.DEFAULT_SCAN_SIZE,
			KeyIconStyle: 0,
			CheckUpdate:  true,
			AllowTrack:   true,
		},
		Proxy: Proxy{
			Enabled:  false,
			Protocal: "",
			Addr:     "",
		},
	}
}

type PreferencesBehavior struct {
	Welcomed        bool `json:"welcomed" yaml:"welcomed"`
	AsideWidth      int  `json:"asideWidth" yaml:"aside_width"`
	WindowWidth     int  `json:"windowWidth" yaml:"window_width"`
	WindowHeight    int  `json:"windowHeight" yaml:"window_height"`
	WindowMaximised bool `json:"windowMaximised" yaml:"window_maximised"`
	WindowPosX      int  `json:"windowPosX" yaml:"window_pos_x"`
	WindowPosY      int  `json:"windowPosY" yaml:"window_pos_y"`
}

type PreferencesGeneral struct {
	Theme           string   `json:"theme" yaml:"theme"`
	Language        string   `json:"language" yaml:"language"`
	Font            string   `json:"font" yaml:"font,omitempty"`
	FontFamily      []string `json:"fontFamily" yaml:"font_family,omitempty"`
	FontSize        int      `json:"fontSize" yaml:"font_size"`
	ScanSize        int      `json:"scanSize" yaml:"scan_size"`
	KeyIconStyle    int      `json:"keyIconStyle" yaml:"key_icon_style"`
	UseSysProxy     bool     `json:"useSysProxy" yaml:"use_sys_proxy,omitempty"`
	UseSysProxyHttp bool     `json:"useSysProxyHttp" yaml:"use_sys_proxy_http,omitempty"`
	CheckUpdate     bool     `json:"checkUpdate" yaml:"check_update"`
	SkipVersion     string   `json:"skipVersion" yaml:"skip_version,omitempty"`
	AllowTrack      bool     `json:"allowTrack" yaml:"allow_track"`
}

type Proxy struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Protocal string `json:"protocal" yaml:"protocal"`
	Addr     string `json:"addr" yaml:"addr"`
	Port     int    `json:"port" yaml:"port"`
}

type Download struct {
	Dir string `json:"dir" yaml:"dir"`
}
