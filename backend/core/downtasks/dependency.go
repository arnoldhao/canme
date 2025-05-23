package downtasks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"CanMe/backend/consts"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/types"
	"CanMe/backend/utils"

	"go.uber.org/zap"
)

// YtdlpConfig yt-dlp config
type YtdlpConfig struct {
	Version string
	BaseURL string
	Latest  bool
}

// getYtdlpPath get or install yt-dlp
func (s *Service) getYtdlpPath(config YtdlpConfig) (string, error) {
	// default config
	if config.Version == "" {
		config.Version = consts.YTDLP_VERSION
	}

	// check ytdlp
	softinfo, err := s.checkYTDLP()
	if err != nil {
		logger.Error("Failed to check ytdlp", zap.Error(err))
	}

	// if installed and do not need update, return
	if softinfo.Available && !config.Latest {
		return softinfo.ExecPath, nil
	}

	// event stage
	processStage := types.DtStageInstalling
	finalStage := types.DtStageInstalled

	// if need update
	if config.Latest {
		// check update
		softinfo, err = s.checkYTDLPUpdate()
		if err != nil {
			logger.Error("Failed to check ytdlp update", zap.Error(err))
			return "", err
		}

		if softinfo.LatestVersion != "" && softinfo.NeedUpdate {
			config.Version = softinfo.LatestVersion
			softinfo.ExecPath = filepath.Join(softinfo.Path, fmt.Sprintf("yt-dlp.%s", config.Version))
			if runtime.GOOS == "windows" {
				softinfo.ExecPath += ".exe"
			}
			// event stage
			processStage = types.DtStageUpdating
			finalStage = types.DtStageUpdated
		} else {
			return softinfo.ExecPath, fmt.Errorf("ytdlp is already the latest version")
		}
	}

	// build download url
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://gh-proxy.com/github.com/yt-dlp/yt-dlp/releases/download/" + config.Version
	}

	// log info: directory/execPath
	logger.Info("ytdlp check result",
		zap.String("directory", softinfo.Path),
		zap.String("execPath", softinfo.ExecPath),
		zap.Bool("available", softinfo.Available))

	// report to event bus, begin to install
	s.eventBus.Broadcast(consts.TopicDowntasksInstalling, &types.DtProgress{
		Stage:     processStage,
		StageInfo: "0%",
	})

	// makesure directory exists
	if err := os.MkdirAll(softinfo.Path, 0o755); err != nil {
		return "", fmt.Errorf("Create directory failed: %w", err)
	}

	// get yt-dlp repo release name
	filename, err := releaseName()
	if err != nil {
		return "", err
	}

	// build download url
	downloadURL := fmt.Sprintf("%s/%s", baseURL, filename)
	// log info: downloadURL
	logger.Info("ytdlp download starting", zap.String("url", downloadURL))

	// download
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("Create download request failed: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Download failed: HTTP %d", resp.StatusCode)
	}

	// create temp file
	tmpFile, err := os.CreateTemp(softinfo.Path, "yt-dlp-*")
	// log info: temp file
	logger.Info("ytdlp temp file created", zap.String("path", tmpFile.Name()))

	if err != nil {
		return "", fmt.Errorf("Create temp file failed: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// write file, show download progress
	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	lastProgress := 0

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, err := tmpFile.Write(buf[:n]); err != nil {
				tmpFile.Close()
				return "", fmt.Errorf("Write file failed: %w", err)
			}
			downloaded += int64(n)

			// update download progress
			if totalSize > 0 {
				progress := int(float64(downloaded) / float64(totalSize) * 100)
				if progress > lastProgress {
					s.eventBus.Broadcast(consts.TopicDowntasksInstalling, &types.DtProgress{
						Stage:      processStage,
						Percentage: float64(progress),
					})
					lastProgress = progress
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			tmpFile.Close()
			return "", fmt.Errorf("Download file failed: %w", err)
		}
	}
	tmpFile.Close()

	// log info: set execute permission
	logger.Info("ytdlp setting permissions", zap.String("path", tmpFile.Name()))

	// set execute permission
	if err := os.Chmod(tmpFile.Name(), 0o755); err != nil {
		return "", fmt.Errorf("Set execute permission failed: %w", err)
	}

	// log info: move to final location
	logger.Info("ytdlp moving to final location",
		zap.String("from", tmpFile.Name()),
		zap.String("to", softinfo.ExecPath))

	// move to final location
	if err := os.Rename(tmpFile.Name(), softinfo.ExecPath); err != nil {
		// if rename failed (maybe because file already exists), try copy
		if err := copyFile(tmpFile.Name(), softinfo.ExecPath); err != nil {
			return "", fmt.Errorf("Move file failed: %w", err)
		}
	}

	s.eventBus.Broadcast(consts.TopicDowntasksInstalling, &types.DtProgress{
		Stage:      finalStage,
		Percentage: float64(100),
	})

	// save config
	newinfo := types.SoftwareInfo{
		Path:          softinfo.Path,
		ExecPath:      softinfo.ExecPath,
		Available:     true,
		Version:       config.Version,
		LatestVersion: config.Version,
		NeedUpdate:    false,
	}
	s.pref.SetYTDLP(newinfo)

	// log info: installed
	logger.Info("ytdlp installation completed", zap.String("path", newinfo.ExecPath))

	return newinfo.ExecPath, nil
}

// copyFile
func copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Open source file failed: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Create destination file failed: %w", err)
	}
	defer destFile.Close()

	// Copy content
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("Copy file content failed: %w", err)
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("Get source file info failed: %w", err)
	}

	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("Set destination file permissions failed: %w", err)
	}

	return nil
}

func (s *Service) checkYTDLP() (config types.SoftwareInfo, err error) {
	softinfo := s.pref.GetDependsYTDLP()
	if softinfo.Path == "" || softinfo.ExecPath == "" {
		// if not exsited, check system
		dir, err := os.UserCacheDir()
		dir = filepath.Join(dir, "go-ytdlp")
		if err != nil {
			return softinfo, err
		}

		name := fmt.Sprintf("yt-dlp.%s", consts.YTDLP_VERSION)
		softinfo.Path = dir
		softinfo.ExecPath = filepath.Join(dir, name) // include dir and file
		if runtime.GOOS == "windows" {
			softinfo.ExecPath += ".exe"
		}
	}

	if runtime.GOOS == "windows" && !strings.HasSuffix(softinfo.ExecPath, ".exe") {
		softinfo.ExecPath += ".exe"
	}

	// update to support yt-dlp version
	if !strings.Contains(softinfo.ExecPath, "yt-dlp.") {
		name := fmt.Sprintf("yt-dlp.%s", consts.YTDLP_VERSION)
		softinfo.ExecPath = filepath.Join(softinfo.Path, name)
	}

	// Check if file exists
	execPath := softinfo.ExecPath
	info, err := os.Stat(execPath)
	if err != nil {
		// save config and return
		softinfo.Available = false
		s.pref.SetYTDLP(softinfo)
		return softinfo, fmt.Errorf("yt-dlp not found in cache directory: %s", softinfo.Path)
	}

	// Windows: check file exists
	if runtime.GOOS == "windows" {
		// save config and return
		softinfo.Available = true
		s.pref.SetYTDLP(softinfo)
		return softinfo, nil
	}

	// Unix: check executable permission
	if info.Mode()&0111 != 0 {
		// save config and return
		softinfo.Available = true
		s.pref.SetYTDLP(softinfo)
		return softinfo, nil
	} else {
		softinfo.Available = false
		// save config and return
		s.pref.SetYTDLP(softinfo)
		return softinfo, fmt.Errorf("yt-dlp exists but not executable: %s", execPath)
	}
}

func (s *Service) checkFFMpeg() (config types.SoftwareInfo, err error) {
	softinfo := s.pref.GetDependsFFMpeg()
	if softinfo.Path == "" || softinfo.ExecPath == "" {
		// if not exsited, check system
		if runtime.GOOS == "windows" {
			// Windows: manually check PATH to avoid command window flash and sandbox issues
			pathEnv := os.Getenv("PATH")
			paths := strings.Split(pathEnv, string(os.PathListSeparator))
			execName := "ffmpeg.exe"
			for _, p := range paths {
				execPath := filepath.Join(p, execName)
				if info, err := os.Stat(execPath); err == nil && !info.IsDir() {
					// save config and return
					softinfo.Path = filepath.Dir(execPath)
					softinfo.ExecPath = execPath
					softinfo.Available = true
					s.pref.SetFFMpeg(softinfo)
					return softinfo, nil
				}
			}
		} else if runtime.GOOS == "darwin" {
			// macOS security policy: can't get PATH, return directly
			return softinfo, fmt.Errorf("ffmpeg's execPath is not set, please set it and retry.")
		} else {
			// Other Unix-like systems: use LookPath
			if execPath, err := exec.LookPath("ffmpeg"); err == nil {
				// save config and return
				softinfo.Path = filepath.Dir(execPath)
				softinfo.ExecPath = execPath
				softinfo.Available = true
				s.pref.SetFFMpeg(softinfo)
				return softinfo, nil
			}
		}
	}

	// Check if file exists
	execPath := softinfo.ExecPath
	info, err := os.Stat(execPath)
	if err != nil {
		// save config and return
		softinfo.Available = false
		s.pref.SetFFMpeg(softinfo)
		return softinfo, fmt.Errorf("ffmpeg not found in cache directory: %s", execPath)
	}

	// Windows: check file exists
	if runtime.GOOS == "windows" {
		// save config and return
		softinfo.Available = true
		s.pref.SetFFMpeg(softinfo)
		return softinfo, nil
	}

	// Unix: check executable permission
	if info.Mode()&0111 != 0 {
		// save config and return
		softinfo.Available = true
		s.pref.SetFFMpeg(softinfo)
		return softinfo, nil
	} else {
		softinfo.Available = false
		// save config and return
		s.pref.SetFFMpeg(softinfo)
		return softinfo, fmt.Errorf("ffmpeg exists but not executable: %s", execPath)
	}
}

func (s *Service) setFFMpeg(execPath string) (types.SoftwareInfo, error) {
	softinfo := s.pref.GetDependsFFMpeg()
	// only avalibale in macOs
	if runtime.GOOS != "darwin" {
		return softinfo, fmt.Errorf("this operation is only alowed in darwin")
	}

	// handle execPath
	if execPath == "" {
		return softinfo, fmt.Errorf("request execPath is empty")
	}

	// trim file name
	dir := filepath.Dir(execPath)
	execPath = filepath.Join(dir, "ffmpeg")
	info, err := os.Stat(execPath)
	if err != nil {
		// save config and return
		s.pref.SetFFMpeg(softinfo)
		return softinfo, fmt.Errorf("ffmpeg not found in cache directory: %s", execPath)
	}

	// Unix: check executable permission
	if info.Mode()&0111 != 0 {
		// save config and return
		softinfo.Path = dir
		softinfo.ExecPath = execPath
		softinfo.Available = true
		s.pref.SetFFMpeg(softinfo)
		return softinfo, nil
	} else {
		softinfo.Available = false
		// save config and return
		s.pref.SetFFMpeg(softinfo)
		return softinfo, fmt.Errorf("ffmpeg exists but not executable: %s", execPath)
	}
}

func releaseName() (filename string, err error) {
	// Select filename based on OS
	switch runtime.GOOS {
	case "darwin":
		filename = "yt-dlp_macos"
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			filename = "yt-dlp_linux"
		case "arm64":
			filename = "yt-dlp_linux_aarch64"
		default:
			err = fmt.Errorf("Unsupported Linux architecture: %s", runtime.GOARCH)
		}
	case "windows":
		filename = "yt-dlp.exe"
	default:
		err = fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}

	return
}

type latestRelease struct {
	Name    string `json:"name"`
	TagName string `json:"tag_name"`
	Url     string `json:"url"`
	HtmlUrl string `json:"html_url"`
}

func (s *Service) checkYTDLPUpdate() (types.SoftwareInfo, error) {
	info := s.pref.GetDependsYTDLP()
	// request latest version
	httpClient := s.proxyClient.HTTPClient()
	res, err := httpClient.Get(consts.YTDLP_CHECK_UPDATE_URL)
	if err != nil {
		return info, fmt.Errorf("failed to check yt-dlp update: %s", err.Error())
	}

	if res.StatusCode != http.StatusOK {
		return info, fmt.Errorf("failed to check yt-dlp update: %s", res.Status)
	}

	var respObj latestRelease
	err = json.NewDecoder(res.Body).Decode(&respObj)
	if err != nil {
		return info, fmt.Errorf("failed to parse yt-dlp update response: %s", err.Error())
	}

	info.LatestVersion = respObj.TagName
	if info.Version == "" {
		info.NeedUpdate = true
		return info, nil
	}

	if info.LatestVersion == "" {
		info.NeedUpdate = false
		return info, nil
	}

	// compare with current version
	current, err := utils.ParseVersion(info.Version)
	if err != nil {
		return info, fmt.Errorf("failed to parse yt-dlp version: %s", err.Error())
	}
	latest, err := utils.ParseVersion(info.LatestVersion)
	if err != nil {
		return info, fmt.Errorf("failed to parse yt-dlp latest version: %s", err.Error())
	}
	if current.Compare(latest) < 0 {
		info.NeedUpdate = true
	} else {
		info.NeedUpdate = false
	}

	return info, nil
}
