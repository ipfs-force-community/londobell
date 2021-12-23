package api

import "github.com/ipfs-force-community/londobell/racailum/segment"

type SegmentDetail struct {
	Active   string
	Info     *segment.Info
	Boundary *segment.Boundary
}
