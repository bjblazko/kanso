// SPDX-License-Identifier: Apache-2.0

package web

import (
	"context"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/bjblazko/kanso/internal/recipeimport"
)

// aiTarget says where text goes when a function uses the AI (ADR-013):
// shown next to every such function as "KI · extern: host" or "KI · im eigenen Netz: host".
type aiTarget struct {
	Host     string
	External bool
}

// aiTarget returns where the AI is, or nil when none is switched on. A host
// that cannot be resolved counts as external.
func (s *Server) aiTarget(ctx context.Context) *aiTarget {
	cfg := s.AI.Load()
	if !cfg.Ready() {
		return nil
	}
	t := &aiTarget{Host: cfg.Host(), External: true}
	if a, err := netip.ParseAddr(t.Host); err == nil {
		t.External = recipeimport.IsPublic(a)
		return t
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", t.Host); err == nil && len(addrs) > 0 {
		t.External = slices.ContainsFunc(addrs, recipeimport.IsPublic)
	}
	return t
}

type aiMarkArgs struct {
	T *aiTarget
	V view
}
