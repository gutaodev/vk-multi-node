package main

import (
	"github.com/gutaodev/vk-multi-node/cmd/virtual-kubelet/internal/provider"
	"github.com/gutaodev/vk-multi-node/cmd/virtual-kubelet/internal/provider/mock"
)

func registerMock(s *provider.Store) {
	s.Register("mock", func(cfg provider.InitConfig) (provider.Provider, error) { //nolint:errcheck
		return mock.NewMockProvider(
			cfg.ConfigPath,
			cfg.NodeName,
			cfg.OperatingSystem,
			cfg.InternalIP,
			cfg.DaemonPort,
		)
	})
}
