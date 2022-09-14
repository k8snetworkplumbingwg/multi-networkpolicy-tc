package driver

type cQDisc struct {
	Kind   string `json:"kind"`
	Handle string `json:"handle"`
	Parent string `json:"parent"`
}

type cChain struct {
	Parent string `json:"parent"`
	Chain  uint16 `json:"chain"`
}

type cFilter struct {
	Protocol string          `json:"protocol"`
	Priority uint16          `json:"pref"`
	Kind     string          `json:"kind"`
	Chain    uint16          `json:"chain"`
	Options  *cFilterOptions `json:"options,omitempty"`
}

type cFilterOptions struct {
	Handle  uint32      `json:"handle"`
	Keys    cFlowerKeys `json:"keys"`
	Actions []cAction   `json:"actions"`
}

type cFlowerKeys struct {
	VlanEthType *string `json:"vlan_ethtype,omitempty"`
	IPProto     *string `json:"ip_proto,omitempty"`
	DstIP       *string `json:"dst_ip,omitempty"`
	DstPort     *uint16 `json:"dst_port,omitempty"`
}

type cAction struct {
	Order         uint           `json:"order"`
	Kind          string         `json:"kind"`
	ControlAction cControlAction `json:"control_action"`
}

type cControlAction struct {
	Type string `json:"type"`
}
