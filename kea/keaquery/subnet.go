package keaquery

import (
	"encoding/json"
	"fmt"
	"net/netip"
)

type subnetQueryKind int

const (
	sqUnknown subnetQueryKind = iota
	sqByID
	sqBySubnet
)

// SubnetQuery is a single type representing either:
//   - { "id": <subnet-id> }
//   - { "subnet": "<prefix>" }
type SubnetQuery struct {
	kind subnetQueryKind

	id     uint32
	subnet netip.Prefix
}

func SubnetByID(id uint32) SubnetQuery {
	return SubnetQuery{
		kind: sqByID,
		id:   id,
	}
}

func SubnetByPrefix(subnet netip.Prefix) SubnetQuery {
	return SubnetQuery{
		kind:   sqBySubnet,
		subnet: subnet,
	}
}

func (q SubnetQuery) ID() (id uint32, ok bool) {
	if q.kind != sqByID {
		return 0, false
	}
	return q.id, true
}

func (q SubnetQuery) Subnet() (subnet netip.Prefix, ok bool) {
	if q.kind != sqBySubnet {
		return netip.Prefix{}, false
	}
	return q.subnet, true
}

func (q SubnetQuery) MarshalJSON() ([]byte, error) {
	switch q.kind {
	case sqByID:
		aux := struct {
			ID uint32 `json:"id"`
		}{
			ID: q.id,
		}
		return json.Marshal(aux)
	case sqBySubnet:
		aux := struct {
			Subnet netip.Prefix `json:"subnet"`
		}{
			Subnet: q.subnet,
		}
		return json.Marshal(aux)
	default:
		return nil, fmt.Errorf("SubnetQuery: invalid variant (neither id nor subnet set)")
	}
}
