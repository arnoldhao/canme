package downtasks

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/browercookies"
	"CanMe/backend/pkg/dependencies"
	"CanMe/backend/pkg/dependencies/providers"
	"CanMe/backend/pkg/downinfo"
	"CanMe/backend/pkg/events"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/pkg/proxy"
	"CanMe/backend/services/preferences"
	"CanMe/backend/storage"
	"CanMe/backend/types"

	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"encoding/json"

	"github.com/lrstanley/go-ytdlp"
	"go.uber.org/zap"
)

// Service 处理视频下载和处理
type Service struct {
	ctx         context.Context
	taskManager *TaskManager
	// 事件总线
	eventBus      events.EventBus
	metadataCache sync.Map // 用于缓存视频元数据
	// proxy
	proxyManager proxy.ProxyManager
	// download
	downloadClient *downinfo.Client
	// pref
	pref *preferences.Service
	// ffmpeg exec path
	ffmpegExecPath string
	// bolt storage
	boltStorage *storage.BoltStorage

	// dependencies
	depManager dependencies.Manager

	// cookie manager
	cookieManager browercookies.CookieManager
}

func NewService(eventBus events.EventBus,
	proxyManager proxy.ProxyManager,
	downloadClient *downinfo.Client,
	pref *preferences.Service,
	boltStorage *storage.BoltStorage,
) *Service {

	// 创建依赖管理器
	depManager := dependencies.NewManager(eventBus, proxyManager, boltStorage)

	// 注册依赖提供者
	depManager.Register(providers.NewYTDLPProvider(eventBus))
	depManager.Register(providers.NewFFmpegProvider(eventBus))

	// 初始化默认依赖信息
	if err := depManager.InitializeDefaultDependencies(); err != nil {
		logger.Error("Failed to initialize default dependencies", zap.Error(err))
	}

	s := &Service{
		taskManager:    nil,
		eventBus:       eventBus,
		proxyManager:   proxyManager,
		downloadClient: downloadClient,
		pref:           pref,
		boltStorage:    boltStorage,
		depManager:     depManager,
		cookieManager:  browercookies.NewCookieManager(boltStorage, depManager),
	}

	return s
}

func (s *Service) SetContext(ctx context.Context) {
	s.ctx = ctx
	s.taskManager = NewTaskManager(ctx, s.boltStorage)
}

func (s *Service) ListTasks() []*types.DtTaskStatus {
	return s.taskManager.ListTasks()
}

func (s *Service) Path() string {
	return s.taskManager.Path()
}

// DeleteTask 删除指定ID的任务
func (s *Service) DeleteTask(id string) error {
	return s.taskManager.DeleteTask(id)
}

func (s *Service) GetTaskStatus(id string) (*types.DtTaskStatus, error) {
	status := s.taskManager.GetTask(id)
	if status == nil {
		return nil, fmt.Errorf("task not found")
	}
	return status, nil
}

func (s *Service) GetTaskStatusByURL(url string) (bool, *types.DtTaskStatus, error) {
	list := s.ListTasks()
	for _, task := range list {
		if task.URL == url {
			return true, task, nil
		}
	}

	return false, nil, nil
}

func (s *Service) GetFormats() map[string][]*types.ConversionFormat {
	return s.taskManager.ListAvalibleConversionFormats()
}

func (s *Service) newCommand(enbaledFFMpeg bool, cookiesFile string) (*ytdlp.Command, error) {
	// new
	dl := ytdlp.New()

	// proxy
	if httpProxy := s.proxyManager.GetProxyString(); httpProxy != "" {
		dl.SetEnvVar("HTTP_PROXY", httpProxy).
			SetEnvVar("HTTPS_PROXY", httpProxy)
	}

	// yt-dlp mustinstall
	ytExecPath, err := s.YTDLPExecPath()
	if err != nil {
		return nil, err
	}
	dl.SetExecutable(ytExecPath)

	// set temp dir
	tempDir, err := s.downDir("temp")
	if err != nil {
		return nil, err
	}

	// set env
	dl.SetEnvVar("TEMP", tempDir)
	dl.SetEnvVar("TMP", tempDir)

	// ffmpeg
	if enbaledFFMpeg {
		ffExecPath, err := s.FFMPEGExecPath()
		if err != nil {
			return nil, err
		}
		// set ffmpeg
		dl.FFmpegLocation(ffExecPath)
	}

	if cookiesFile != "" {
		// set cookies
		dl.Cookies(cookiesFile)
	}

	return dl, nil
}

// ParseURL 从 YouTube 获取视频内容信息
func (s *Service) ParseURL(url string, browser string) (*ytdlp.ExtractedInfo, error) {
	// 获取Cookies
	var cookiesFile string
	if browser != "" {
		netscapecookies, err := s.GetNetscapeCookiesByDomain(browser, url)
		if err == nil && netscapecookies != "" {
			// 保存netscapecookies为文件
			path, err := s.YTDLPPath()
			if err != nil {
				return nil, err
			}
			cookiesFile = filepath.Join(path, "cookies.txt")
			err = os.WriteFile(cookiesFile, []byte(netscapecookies), 0644)
			if err != nil {
				return nil, err
			}

			defer os.Remove(cookiesFile)
		}
	}
	// 创建 yt-dlp 命令构建器
	dl, err := s.newCommand(false, cookiesFile)
	if err != nil {
		return nil, err
	}

	// 添加选项
	dl.SkipDownload()   // 不下载视频，只获取信息
	dl.DumpSingleJSON() // 使用 DumpSingleJSON 获取结构化的 JSON 输出

	// 运行 yt-dlp 命令
	result, err := dl.Run(s.ctx, url)
	if err != nil {
		return nil, err
	}

	// 解析 JSON 数据为 ExtractedInfo 结构体
	var info ytdlp.ExtractedInfo
	if err := json.Unmarshal([]byte(result.Stdout), &info); err != nil {
		return nil, err
	}

	// 缓存元数据
	s.cacheMetadata(url, &info)

	return &info, nil
}

func (s *Service) getVideoMetadata(url, browser string) (*ytdlp.ExtractedInfo, error) {
	// 尝试从缓存获取元数据
	metadata, ok := s.getCachedMetadata(url)
	if ok {
		return metadata, nil
	}

	// 如果缓存中没有，则重新获取
	return s.ParseURL(url, browser)
}

type InfoChan chan *types.FillTaskInfo

// ProgressChan is a channel for receiving download progress updates
type ProgressChan chan *types.DtProgress

// Download 开始视频下载和处理流程
func (s *Service) Download(request *types.DtDownloadRequest) (*types.DtDownloadResponse, error) {
	// 创建新任务
	taskID := uuid.New().String()
	task := s.taskManager.CreateTask(taskID)

	task.Type = consts.TASK_TYPE_CUSTOM

	// 尝试从缓存获取元数据
	metadata, err := s.getVideoMetadata(request.URL, request.Browser)
	if err != nil {
		s.handleTaskError(task, err, nil)
		return nil, err
	}

	// request params
	task.DownloadSubs = request.DownloadSubs
	task.SubLangs = request.SubLangs
	task.SubFormat = request.SubFormat
	task.TranslateTo = request.TranslateTo
	task.SubtitleStyle = request.SubtitleStyle

	// recode info
	task.RecodeFormatNumber = request.RecodeFormatNumber
	if request.RecodeFormatNumber != 0 {
		recodeExt, err := s.taskManager.GetConversionFormatExtension(request.RecodeFormatNumber)
		if err != nil {
			// ignore
		} else {
			task.RecodeExtention = recodeExt
		}
	}

	// core metadata
	task.Extractor = *metadata.Extractor
	task.Title = *metadata.Title
	task.Thumbnail = *metadata.Thumbnail
	task.URL = request.URL
	task.Stage = types.DtStageDownloading
	task.Percentage = 0
	task.FormatID = request.FormatID

	// 兼容Bilibili番剧
	if metadata.Uploader != nil {
		task.Uploader = *metadata.Uploader
	} else if metadata.Series != nil {
		task.Uploader = *metadata.Series
	} else {
		task.Uploader = *metadata.Extractor // default
	}
	task.Duration = *metadata.Duration

	// 获取输出目录
	outputDir, err := s.downDir(task.Extractor)
	if err == nil {
		task.OutputDir = outputDir
	}

	// 如果有格式信息，设置格式
	if formats := metadata.Formats; formats != nil {
		for _, format := range formats {
			if *format.FormatID == request.FormatID {
				task.Format = *format.Extension
				// file size
				if format.FileSizeApprox != nil {
					task.FileSize = int64(*format.FileSizeApprox)
				} else if format.FileSize != nil {
					task.FileSize = int64(*format.FileSize)
				} else {
					task.FileSize = 0
				}

				// quality
				if format.Resolution != nil {
					task.Resolution = *format.Resolution
				} else if format.Height != nil && format.Width != nil {
					task.Resolution = fmt.Sprintf("%v x %v", *format.Width, *format.Height)
				} else {
					task.Resolution = "Unknown"
				}
				break
			}
		}
	}

	s.taskManager.UpdateTask(task)

	resp := &types.DtDownloadResponse{
		ID:     taskID,
		Status: types.DtStageDownloading,
	}

	// initial task info channel
	infoChan := make(InfoChan, 1)

	// 初始化进度通道
	progressChan := make(ProgressChan, 100)

	// 启动处理流程
	go s.processTask(task, &types.DownloadVideoRequest{
		Type:          task.Type,
		URL:           request.URL,
		Browser:       request.Browser,
		FormatID:      request.FormatID,
		DownloadSubs:  request.DownloadSubs,
		SubLangs:      request.SubLangs,
		SubFormat:     request.SubFormat,
		TranslateTo:   request.TranslateTo,
		SubtitleStyle: request.SubtitleStyle,
	}, infoChan, progressChan)

	// start info monitor
	go s.fillTaskInfo(infoChan)

	// 启动进度监控
	go s.monitorProgress(progressChan)

	return resp, nil
}

// QuickDownload 快速下载视频
func (s *Service) QuickDownload(request *types.DtQuickDownloadRequest) (*types.DtQuickDownloadResponse, error) {
	// 创建新任务
	taskID := uuid.New().String()
	task := s.taskManager.CreateTask(taskID)

	task.Type = request.Type
	task.URL = request.URL
	task.Browser = request.Browser

	task.Stage = types.DtStageDownloading
	task.Percentage = 0

	// recode info
	task.RecodeFormatNumber = request.RecodeFormatNumber
	if request.RecodeFormatNumber != 0 {
		recodeExt, err := s.taskManager.GetConversionFormatExtension(request.RecodeFormatNumber)
		if err != nil {
			// ignore
		} else {
			task.RecodeExtention = recodeExt
		}
	}

	// define output dir, quick / mcp
	outputDir, err := s.downDir(request.Type)
	if err == nil {
		task.OutputDir = outputDir
	}

	s.taskManager.UpdateTask(task)

	resp := &types.DtQuickDownloadResponse{
		ID:     taskID,
		Status: types.DtStageDownloading,
	}

	// initial task info channel
	infoChan := make(InfoChan, 1)

	// 初始化进度通道
	progressChan := make(ProgressChan, 100)

	// 启动处理流程
	go s.processTask(task, &types.DownloadVideoRequest{
		Type:        request.Type,
		URL:         request.URL,
		Browser:     request.Browser,
		Video:       request.Video,
		BestCaption: request.BestCaption,
	}, infoChan, progressChan)

	// start info monitor
	go s.fillTaskInfo(infoChan)

	// 启动进度监控
	go s.monitorProgress(progressChan)

	return resp, nil
}

// 缓存元数据
func (s *Service) cacheMetadata(url string, metadata *ytdlp.ExtractedInfo) {
	s.metadataCache.Store(url, metadata)
}

// 获取缓存的元数据
func (s *Service) getCachedMetadata(url string) (*ytdlp.ExtractedInfo, bool) {
	value, ok := s.metadataCache.Load(url)
	if !ok {
		return nil, false
	}
	metadata, ok := value.(*ytdlp.ExtractedInfo)
	return metadata, ok
}

// processTask 处理任务的主流程
func (s *Service) processTask(task *types.DtTaskStatus, request *types.DownloadVideoRequest, infoChan InfoChan, progressChan ProgressChan) {
	defer close(progressChan)

	// 第一阶段：下载视频
	err := s.downloadVideo(task, request, infoChan, progressChan)
	if err != nil {
		s.handleTaskError(task, err, progressChan)
		return
	}
	s.taskManager.UpdateTask(task)

	// 第二阶段：翻译字幕（如果需要）
	if request.Type == consts.TASK_TYPE_CUSTOM {
		if request.DownloadSubs && request.TranslateTo != "" {
			task.Stage = types.DtStageTranslating
			s.taskManager.UpdateTask(task)

			// 发送阶段变更通知
			progressChan <- &types.DtProgress{
				ID:         task.ID,
				Type:       task.Type,
				Stage:      types.DtStageTranslating,
				Percentage: 0,
				StageInfo:  "Start translating subtitles",
			}

			subtitleFile, err := s.translateSubtitles(task, progressChan)
			if err != nil {
				s.handleTaskError(task, err, progressChan)
				return
			}
			task.TranslatedSubs = append(task.TranslatedSubs, subtitleFile)
			// add to all files
			task.AllFiles = append(task.AllFiles, subtitleFile)
			s.taskManager.UpdateTask(task)
		} else {
			task.TranslatedSubs = []string{}
		}

		// 第三阶段：嵌入字幕（如果需要）
		if request.DownloadSubs && request.TranslateTo != "" {
			task.Stage = types.DtStageEmbedding
			s.taskManager.UpdateTask(task)

			// 发送阶段变更通知
			progressChan <- &types.DtProgress{
				ID:         task.ID,
				Type:       task.Type,
				Stage:      types.DtStageEmbedding,
				Percentage: 0,
				StageInfo:  "Start embedding subtitles",
			}

			embeddedVideo, err := s.embedSubtitles(task, progressChan)
			if err != nil {
				s.handleTaskError(task, err, progressChan)
				return
			}
			task.EmbeddedVideoFiles = append(task.EmbeddedVideoFiles, embeddedVideo)
			// add to all files
			task.AllFiles = append(task.AllFiles, embeddedVideo)
			s.taskManager.UpdateTask(task)
		} else {
			task.EmbeddedVideoFiles = []string{}
		}
	}
	// 完成所有处理
	task.Stage = types.DtStageCompleted
	s.taskManager.UpdateTask(task)

	// 发送完成通知
	progressChan <- &types.DtProgress{
		ID:         task.ID,
		Type:       task.Type,
		Stage:      types.DtStageCompleted,
		Percentage: 100,
		StageInfo:  "Processing completed",
	}
}

// handleTaskError 处理任务错误
func (s *Service) handleTaskError(task *types.DtTaskStatus, err error, progressChan ProgressChan) {
	task.Stage = types.DtStageFailed
	task.Error = err.Error()
	s.taskManager.UpdateTask(task)

	progressChan <- &types.DtProgress{
		ID:         task.ID,
		Type:       task.Type,
		Stage:      types.DtStageFailed,
		Error:      err.Error(),
		Percentage: 0,
		StageInfo:  "Processing failed",
	}
}

// downloadVideo 实现视频下载阶段
func (s *Service) downloadVideo(task *types.DtTaskStatus, request *types.DownloadVideoRequest, infoChan InfoChan, progressChan ProgressChan) error {
	// 发送阶段开始通知
	progressChan <- &types.DtProgress{
		ID:         task.ID,
		Type:       task.Type,
		Stage:      types.DtStageDownloading,
		Percentage: 0,
		StageInfo:  "Start downloading video",
	}

	// 获取Cookies
	var cookiesFile string
	if request.Browser != "" {
		netscapecookies, err := s.GetNetscapeCookiesByDomain(request.Browser, request.URL)
		if err == nil && netscapecookies != "" {
			// 保存netscapecookies为文件
			path, err := s.YTDLPPath()
			if err != nil {
				return err
			}
			cookiesFile = filepath.Join(path, "cookies.txt")
			err = os.WriteFile(cookiesFile, []byte(netscapecookies), 0644)
			if err != nil {
				return err
			}

			defer os.Remove(cookiesFile)
		}
	}

	dl, err := s.newCommand(true, cookiesFile)
	if err != nil {
		s.handleTaskError(task, err, progressChan)
		return err
	}

	if task.Type == "custom" {
		metadata, err := s.getVideoMetadata(request.URL, request.Browser)
		if err != nil {
			s.handleTaskError(task, err, progressChan)
			return err
		}

		// 检查请求的 format_id 是否存在 VCodec 且不存在 ACodes的情况，这种需要增加 bestaudio
		var videoExt string
		if request.FormatID != "" {
			needAudio := false
			for _, format := range metadata.Formats {
				if format.FormatID != nil && *format.FormatID == request.FormatID {
					if format.VCodec != nil && *format.VCodec != "none" {
						if format.ACodec == nil || *format.ACodec == "none" {
							needAudio = true
						}
					}

					if format.Extension != nil {
						videoExt = *format.Extension
					}
					break
				}
			}

			if needAudio {
				if videoExt == "mp4" {
					// MP4 视频，使用 M4A 音频
					dl.Format(request.FormatID + "+bestaudio[ext=m4a]")
					dl.MergeOutputFormat("mp4")
				} else if videoExt == "webm" {
					// WebM 视频，使用 WebM 音频
					dl.Format(request.FormatID + "+bestaudio[ext=webm]")
					dl.MergeOutputFormat("webm")
				} else {
					// 其他情况，让 yt-dlp 自行决定
					dl.Format(request.FormatID + "+bestaudio")
					// 保持原来的设置
					dl.MergeOutputFormat("mp4/webm")
				}
			} else {
				dl.Format(request.FormatID)
			}
		} else {
			dl.UnsetFormat()
		}

		if request.DownloadSubs {
			dl.WriteSubs() // 启用字幕下载

			// 如果指定了字幕语言
			if len(request.SubLangs) > 0 {
				dl.SubLangs(strings.Join(request.SubLangs, ","))
			} else {
				dl.SubLangs("all") // 下载所有可用字幕
			}

			// 如果指定了字幕格式
			if request.SubFormat != "" {
				dl.SubFormat(request.SubFormat)
			} else {
				dl.SubFormat("best") // 使用最佳字幕格式
			}
		}

	} else { // if type == quick || mcp
		// format
		if request.Video != "" {
			switch request.Video {
			case "best":
				dl.UnsetFormat()
			default:
				dl.Format(request.Video)
			}
		}

		// caption
		if request.BestCaption {
			dl.SubFormat("best")
		}
	}

	// 设置工作目录和输出文件
	dl.SetWorkDir(task.OutputDir).
		NoPlaylist().
		NoOverwrites().
		Output("%(title)s_%(height)sp_%(fps)dfps.%(ext)s")

	// Recode
	if task.RecodeExtention != "" {
		dl.RecodeVideo(task.RecodeExtention)
	}

	var once sync.Once

	// 设置进度回调
	dl.ProgressFunc(time.Second, func(update ytdlp.ProgressUpdate) {
		once.Do(func() {
			infoChan <- &types.FillTaskInfo{
				ID:   task.ID,
				Info: update.Info,
			}
		})

		// todo:解决为什么同时下载字幕的YouTube视频，会仅推送下载字幕update，而不推送下载视频update
		progress := &types.DtProgress{
			ID:            task.ID,
			Type:          task.Type,
			Stage:         types.DtStageDownloading,
			Percentage:    update.Percent(),
			Speed:         fmt.Sprintf("%.2f MB/s", float64(update.DownloadedBytes)/update.Duration().Seconds()/1024/1024),
			Downloaded:    fmt.Sprintf("%.2f MB", float64(update.DownloadedBytes)/1024/1024),
			TotalSize:     fmt.Sprintf("%.2f MB", float64(update.TotalBytes)/1024/1024),
			EstimatedTime: formatDuration(update.ETA()),
		}

		select {
		case progressChan <- progress:
		case <-s.ctx.Done():
			progress.Stage = types.DtStageCancelled
			progress.Error = "download cancelled"
			progressChan <- progress
			return
		default:
			// Channel is full, skip this update
		}
	})

	// 执行下载
	result, err := dl.Run(s.ctx, request.URL)
	if err != nil {
		s.handleTaskError(task, err, progressChan)
		return fmt.Errorf("Download video failed: %w", err)
	}

	// 解析下载结果
	s.parseYtdlpOutput(task, result)

	return nil
}

func (s *Service) fillTaskInfo(infoChan InfoChan) {
	for taskInfo := range infoChan {
		if taskInfo == nil {
			continue
		}

		task := s.taskManager.GetTask(taskInfo.ID)
		if task != nil {
			// core metadata
			task.Extractor = *taskInfo.Info.Extractor
			task.Title = *taskInfo.Info.Title
			task.Thumbnail = *taskInfo.Info.Thumbnail

			// 兼容Bilibili番剧
			if taskInfo.Info.Uploader != nil {
				task.Uploader = *taskInfo.Info.Uploader
			} else if taskInfo.Info.Series != nil {
				task.Uploader = *taskInfo.Info.Series
			} else {
				task.Uploader = *taskInfo.Info.Extractor // default
			}
			task.Duration = *taskInfo.Info.Duration

			if task.Format == "" {
				// format extension
				task.Format = taskInfo.Info.Extension

				// file size
				if taskInfo.Info.FileSizeApprox != nil {
					task.FileSize = int64(*taskInfo.Info.FileSizeApprox)
				} else if taskInfo.Info.FileSize != nil {
					task.FileSize = int64(*taskInfo.Info.FileSize)
				} else {
					task.FileSize = 0
				}

				// quality
				if taskInfo.Info.Resolution != nil {
					task.Resolution = *taskInfo.Info.Resolution
				} else if taskInfo.Info.Height != nil && taskInfo.Info.Width != nil {
					task.Resolution = fmt.Sprintf("%v x %v", *taskInfo.Info.Width, *taskInfo.Info.Height)
				} else {
					task.Resolution = "Unknown"
				}

				// update
				s.taskManager.UpdateTask(task)

				// 创建事件
				event := &events.BaseEvent{
					ID:        uuid.New().String(),
					Type:      consts.TopicDowntasksInstalling,
					Source:    "downtasks",
					Timestamp: time.Now(),
					Data: &types.DTSignal{
						ID:      task.ID,
						Type:    task.Type,
						Stage:   task.Stage,
						Refresh: true,
					},
					Metadata: map[string]interface{}{
						"task": task,
					},
				}

				s.eventBus.Publish(s.ctx, event)
			}
		}
	}
}

// parseYtdlpOutput 解析yt-dlp的输出结果，提取最终保存的文件信息
func (s *Service) parseYtdlpOutput(task *types.DtTaskStatus, result *ytdlp.Result) {
	tempFiles := []string{}
	// 按行解析输出
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		// 检测下载目标文件
		if strings.Contains(line, "[download] Destination:") {
			filename := strings.TrimPrefix(line, "[download] Destination: ")
			tempFiles = append(tempFiles, filename)
		}

		// 检测合并文件
		if strings.Contains(line, "[Merger] Merging formats into") {
			filenameWithQuotes := strings.TrimPrefix(line, "[Merger] Merging formats into ")
			filename := strings.TrimPrefix(filenameWithQuotes, "\"")
			filename = strings.TrimSuffix(filename, "\"")
			tempFiles = append(tempFiles, filename)
			// merged file is final video
			task.VideoFiles = append(task.VideoFiles, filename)
		}

		// 检测删除的临时文件
		if strings.Contains(line, "Deleting original file") {
			filenameWithQuotes := strings.TrimPrefix(line, "Deleting original file ")
			filename := strings.TrimSuffix(filenameWithQuotes, " (pass -k to keep)")
			for i, file := range tempFiles {
				if file == filename {
					// 从tempFiles中删除
					tempFiles = append(tempFiles[:i], tempFiles[i+1:]...)
				}
			}
		}
	}

	// 更新任务的Files映射
	// 如果有合并文件，将其设为主视频文件
	if len(tempFiles) > 0 {
		task.AllDownloadedFiles = tempFiles
		for _, filename := range tempFiles {
			// detect file type
			fileType := s.detectFileType(filename)
			if fileType == "subtitle" {
				task.SubtitleFiles = append(task.SubtitleFiles, filename)
			} else {
				task.VideoFiles = append(task.VideoFiles, filename)
			}
		}
	} else {
		task.AllDownloadedFiles = []string{}
		task.SubtitleFiles = []string{}
		task.VideoFiles = []string{}
	}
	// add to all files
	task.AllFiles = append(task.AllFiles, task.AllDownloadedFiles...)
	// 更新任务
	s.taskManager.UpdateTask(task)

	// 创建事件
	event := &events.BaseEvent{
		ID:        uuid.New().String(),
		Type:      consts.TopicDowntasksInstalling,
		Source:    "downtasks",
		Timestamp: time.Now(),
		Data: &types.DTSignal{
			ID:      task.ID,
			Type:    task.Type,
			Stage:   task.Stage,
			Refresh: true,
		},
		Metadata: map[string]interface{}{
			"task": task,
		},
	}

	s.eventBus.Publish(s.ctx, event)
}

// translateSubtitles 实现字幕翻译阶段
func (s *Service) translateSubtitles(task *types.DtTaskStatus, progressChan ProgressChan) (string, error) {
	// TODO
	_ = progressChan
	// 返回翻译后的字幕文件路径
	translatedSubFile := fmt.Sprintf("downloads/%s.translated.srt", task.ID)

	// 在实际实现中，这里应该调用翻译API并保存翻译后的字幕

	return translatedSubFile, nil
}

// detectFileType 根据文件名判断文件类型
func (s *Service) detectFileType(filename string) string {
	// 检查是否是字幕文件
	subtitleExts := []string{".vtt", ".srt", ".ass", ".ssa"}
	for _, ext := range subtitleExts {
		if strings.HasSuffix(filename, ext) {
			return "subtitle"
		}
	}

	// 默认为视频文件
	return "video"
}

// embedSubtitles 实现字幕嵌入阶段
func (s *Service) embedSubtitles(task *types.DtTaskStatus, progressChan ProgressChan) (string, error) {
	// TODO
	_ = progressChan
	// 返回最终视频文件路径
	embeddedVideo := fmt.Sprintf("downloads/%s.embedded.mp4", task.ID)

	// 在实际实现中，这里应该调用FFmpeg等工具嵌入字幕

	return embeddedVideo, nil
}

// monitorProgress 监控进度并发送到前端
func (s *Service) monitorProgress(progressChan ProgressChan) {
	for progress := range progressChan {
		if progress == nil {
			continue
		}

		// 根据不同阶段和状态打印不同的日志
		switch progress.Stage {
		case types.DtStageDownloading:
			fmt.Printf("[%s] Downloading: %.2f%%, Speed: %s, Estimated Time: %s\n",
				progress.ID,
				progress.Percentage,
				progress.Speed,
				progress.EstimatedTime,
			)
		case types.DtStageTranslating:
			fmt.Printf("[%s] Translating: %.2f%%, %s\n",
				progress.ID,
				progress.Percentage,
				progress.StageInfo,
			)
		case types.DtStageEmbedding:
			fmt.Printf("[%s] Embedding: %.2f%%, %s\n",
				progress.ID,
				progress.Percentage,
				progress.StageInfo,
			)
		case types.DtStageCompleted:
			fmt.Printf("[%s] Completed\n",
				progress.ID,
			)
		case types.DtStageFailed:
			fmt.Printf("[%s] Failed: %s\n",
				progress.ID,
				progress.Error,
			)
		}

		task := s.taskManager.GetTask(progress.ID)
		if task != nil {
			// update task
			task.UpdateFromProgress(progress)
			s.taskManager.UpdateTask(task)
		}

		// eventbus
		// 创建事件
		event := &events.BaseEvent{
			ID:        uuid.New().String(),
			Type:      consts.TopicDowntasksProgress,
			Source:    "downtasks",
			Timestamp: time.Now(),
			Data:      progress,
			Metadata: map[string]interface{}{
				"progress": progress,
			},
		}

		s.eventBus.Publish(s.ctx, event)
	}
}

func (s *Service) downDir(source string) (string, error) {
	canmeDir := s.downloadClient.GetDownloadDirWithCanMe()
	if canmeDir == "" {
		return "", fmt.Errorf("canme dir is empty")
	}

	dir := filepath.Join(canmeDir, source)
	// check if source dir is exsited
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// if source dir is not exsited, create it
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create source directory: %w", err)
		}
	}

	return dir, nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second) // 四舍五入到最接近的秒

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// Close 关闭服务，清理资源
func (s *Service) Close() error {
	// 关闭任务管理器，确保持久化存储正确关闭
	if s.taskManager != nil {
		return s.taskManager.Close()
	}
	return nil
}
