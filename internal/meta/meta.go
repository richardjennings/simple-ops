package meta

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/richardjennings/simple-ops/internal/manifest/matcher"
)

type (
	Svc struct {
		c *cfg.Svc
		m *manifest.Svc
		i *matcher.Svc
	}
)

func NewSvc(c *cfg.Svc, m *manifest.Svc, i *matcher.Svc) *Svc {
	return &Svc{c: c, m: m, i: i}
}
