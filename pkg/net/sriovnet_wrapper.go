package net

import (
	"github.com/Mellanox/sriovnet"
)

// SriovnetProvider is a wrapper interface on top of sriovnet
type SriovnetProvider interface {
	// GetVfRepresentor gets an uplink netdev and VF index and returns the VF representor
	GetVfRepresentor(uplink string, vfIndex int) (string, error)
	// GetVfIndexByPciAddress gets a VF PCI address (e.g '0000:03:00.4') and
	// returns the correlate VF index.
	GetVfIndexByPciAddress(vfPciAddress string) (int, error)
	// GetUplinkRepresentor gets a VF or PF PCI address (e.g '0000:03:00.4') and
	// returns the uplink represntor netdev name for that VF or PF.
	GetUplinkRepresentor(pciAddress string) (string, error)
}

func NewSriovnetProviderImpl() *SriovnetProviderImpl {
	return &SriovnetProviderImpl{}
}

type SriovnetProviderImpl struct{}

func (s *SriovnetProviderImpl) GetVfRepresentor(uplink string, vfIndex int) (string, error) {
	return sriovnet.GetVfRepresentor(uplink, vfIndex)
}

func (s *SriovnetProviderImpl) GetVfIndexByPciAddress(vfPciAddress string) (int, error) {
	return sriovnet.GetVfIndexByPciAddress(vfPciAddress)
}

func (s *SriovnetProviderImpl) GetUplinkRepresentor(pciAddress string) (string, error) {
	return sriovnet.GetUplinkRepresentor(pciAddress)
}
