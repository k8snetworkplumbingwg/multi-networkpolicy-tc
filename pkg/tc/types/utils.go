package types

// compare first with second. They are equal if:
//  1. first and second point to the same address (nil or otherwise)
//  2. first and second contain the same value
//  3. if nilVal != nil
//     3.1 first is not nil and *nilVal equals to *first
//     3.2 second is not nil and *nilVal equals to *second
func compare[C comparable](first *C, second *C, nilVal *C) bool {
	if first == second {
		return true
	}

	if first != nil && second != nil {
		return *first == *second
	}

	if nilVal != nil {
		if first != nil && *first == *nilVal {
			return true
		}
		if second != nil && *second == *nilVal {
			return true
		}
	}
	return false
}

// ProtoToVlanProto converts FilterProtocol to VLAN ethtype, returns "" if conversion is invalid.
func ProtoToVlanProto(proto FilterProtocol) string {
	var vlanProto string

	switch proto {
	case FilterProtocolIPv4:
		vlanProto = "ip"
	case FilterProtocolIPv6:
		vlanProto = "ipv6"
	}
	return vlanProto
}
