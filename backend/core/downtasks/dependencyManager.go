package downtasks

import (
	"CanMe/backend/types"
)

// YTDLPExecPath 获取 yt-dlp 可执行文件路径
func (s *Service) YTDLPExecPath() (string, error) {
	return s.executablePath(types.DependencyYTDLP)
}

// FFMPEGExecPath 获取 FFmpeg 可执行文件路径
func (s *Service) FFMPEGExecPath() (string, error) {
	return s.executablePath(types.DependencyFFmpeg)
}

// InstallDependency 安装依赖
func (s *Service) InstallDependency(depType types.DependencyType, config types.DownloadConfig) (*types.DependencyInfo, error) {
	return s.depManager.Install(s.ctx, depType, config)
}

// UpdateDependencyWithMirror 使用指定镜像更新依赖
func (s *Service) UpdateDependencyWithMirror(depType types.DependencyType, config types.DownloadConfig) (*types.DependencyInfo, error) {
	return s.depManager.UpdateWithMirror(s.ctx, depType, config)
}

// ListDependencies 列出所有依赖
func (s *Service) ListDependencies() (map[types.DependencyType]*types.DependencyInfo, error) {
	return s.depManager.List(s.ctx)
}

// CheckDependencyUpdates 检查依赖更新
func (s *Service) CheckDependencyUpdates() (map[types.DependencyType]*types.DependencyInfo, error) {
	return s.depManager.CheckUpdates(s.ctx)
}

// DependenciesReady 检查所有依赖是否已准备好
func (s *Service) DependenciesReady() (bool, error) {
	return s.depManager.DependenciesReady(s.ctx)
}

// ValidateDependencies 验证所有依赖可用性
func (s *Service) ValidateDependencies() error {
	return s.depManager.ValidateDependencies(s.ctx)
}

// executablePath 获取可执行文件路径
func (s *Service) executablePath(depType types.DependencyType) (string, error) {
	info, err := s.depManager.Get(s.ctx, depType)
	if err != nil {
		return "", err
	}

	return info.ExecPath, nil
}
