package config

import (
	"github.com/pbnjay/memory"
)

const (
	tekuTagTest         string = "consensys/teku:24.1.0"
	tekuTagProd         string = "consensys/teku:24.1.0"
	defaultTekuMaxPeers uint16 = 100
)

// Configuration for Teku
type TekuConfig struct {
	Title string `yaml:"-"`

	// Common parameters that Teku doesn't support and should be hidden
	UnsupportedCommonParams []string `yaml:"-"`

	// Max number of P2P peers to connect to
	JvmHeapSize Parameter `yaml:"jvmHeapSize,omitempty"`

	// The max number of P2P peers to connect to
	MaxPeers Parameter `yaml:"maxPeers,omitempty"`

	// The archive mode flag
	ArchiveMode Parameter `yaml:"archiveMode,omitempty"`

	// The Docker Hub tag for Lighthouse
	ContainerTag Parameter `yaml:"containerTag,omitempty"`

	// Custom command line flags for the BN
	AdditionalBnFlags Parameter `yaml:"additionalBnFlags,omitempty"`

	// Custom command line flags for the VC
	AdditionalVcFlags Parameter `yaml:"additionalVcFlags,omitempty"`
}

// Generates a new Teku configuration
func NewTekuConfig(cfg *RocketPoolConfig) *TekuConfig {
	return &TekuConfig{
		Title: "Teku Settings",

		UnsupportedCommonParams: []string{},

		JvmHeapSize: Parameter{
			ID:                 "jvmHeapSize",
			Name:               "JVM Heap Size",
			Description:        "The max amount of RAM, in MB, that Teku's JVM should limit itself to. Setting this lower will cause Teku to use less RAM, though it will always use more than this limit.\n\nUse 0 for automatic allocation.",
			Type:               ParameterType_Uint,
			Default:            map[Network]interface{}{Network_All: getTekuHeapSize()},
			AffectsContainers:  []ContainerID{ContainerID_Eth1},
			CanBeBlank:         false,
			OverwriteOnUpgrade: false,
		},

		MaxPeers: Parameter{
			ID:                 "maxPeers",
			Name:               "Max Peers",
			Description:        "The maximum number of peers your client should try to maintain. You can try lowering this if you have a low-resource system or a constrained network.",
			Type:               ParameterType_Uint16,
			Default:            map[Network]interface{}{Network_All: defaultTekuMaxPeers},
			AffectsContainers:  []ContainerID{ContainerID_Eth2},
			CanBeBlank:         false,
			OverwriteOnUpgrade: false,
		},

		ArchiveMode: Parameter{
			ID:                 "archiveMode",
			Name:               "Enable Archive Mode",
			Description:        "When enabled, Teku will run in \"archive\" mode which means it can recreate the state of the Beacon chain for a previous block. This is required for manually generating the Merkle rewards tree.\n\nIf you are sure you will never be manually generating a tree, you can disable archive mode.",
			Type:               ParameterType_Bool,
			Default:            map[Network]interface{}{Network_All: false},
			AffectsContainers:  []ContainerID{ContainerID_Eth2},
			CanBeBlank:         false,
			OverwriteOnUpgrade: false,
		},

		ContainerTag: Parameter{
			ID:          "containerTag",
			Name:        "Container Tag",
			Description: "The tag name of the Teku container you want to use on Docker Hub.",
			Type:        ParameterType_String,
			Default: map[Network]interface{}{
				Network_Mainnet: tekuTagProd,
				Network_Prater:  tekuTagTest,
				Network_Devnet:  tekuTagTest,
				Network_Holesky: tekuTagTest,
			},
			AffectsContainers:  []ContainerID{ContainerID_Eth2, ContainerID_Validator},
			CanBeBlank:         false,
			OverwriteOnUpgrade: true,
		},

		AdditionalBnFlags: Parameter{
			ID:                 "additionalBnFlags",
			Name:               "Additional Beacon Node Flags",
			Description:        "Additional custom command line flags you want to pass Teku's Beacon Node, to take advantage of other settings that the Smartnode's configuration doesn't cover.",
			Type:               ParameterType_String,
			Default:            map[Network]interface{}{Network_All: ""},
			AffectsContainers:  []ContainerID{ContainerID_Eth2},
			CanBeBlank:         true,
			OverwriteOnUpgrade: false,
		},

		AdditionalVcFlags: Parameter{
			ID:                 "additionalVcFlags",
			Name:               "Additional Validator Client Flags",
			Description:        "Additional custom command line flags you want to pass Teku's Validator Client, to take advantage of other settings that the Smartnode's configuration doesn't cover.",
			Type:               ParameterType_String,
			Default:            map[Network]interface{}{Network_All: ""},
			AffectsContainers:  []ContainerID{ContainerID_Validator},
			CanBeBlank:         true,
			OverwriteOnUpgrade: false,
		},
	}
}

// Get the parameters for this config
func (cfg *TekuConfig) GetParameters() []*Parameter {
	return []*Parameter{
		&cfg.JvmHeapSize,
		&cfg.MaxPeers,
		&cfg.ArchiveMode,
		&cfg.ContainerTag,
		&cfg.AdditionalBnFlags,
		&cfg.AdditionalVcFlags,
	}
}

// Get the recommended heap size for Teku
func getTekuHeapSize() uint64 {
	totalMemoryGB := memory.TotalMemory() / 1024 / 1024 / 1024
	if totalMemoryGB < 9 {
		return 2048
	}
	return 0
}

// Get the common params that this client doesn't support
func (cfg *TekuConfig) GetUnsupportedCommonParams() []string {
	return cfg.UnsupportedCommonParams
}

// Get the Docker container name of the validator client
func (cfg *TekuConfig) GetValidatorImage() string {
	return cfg.ContainerTag.Value.(string)
}

// Get the Docker container name of the beacon client
func (cfg *TekuConfig) GetBeaconNodeImage() string {
	return cfg.ContainerTag.Value.(string)
}

// Get the name of the client
func (cfg *TekuConfig) GetName() string {
	return "Teku"
}

// The the title for the config
func (cfg *TekuConfig) GetConfigTitle() string {
	return cfg.Title
}
